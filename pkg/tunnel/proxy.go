package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/armon/go-socks5"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"golang.org/x/crypto/ssh"
)

type SSMConfig struct {
	InstanceID string   `json:"instanceId"`
	Region     string   `json:"region"`
	Subnets    []string `json:"subnets"`
}

type SSHConfig struct {
	Host       string   `json:"host"`
	Username   string   `json:"username"`
	PrivateKey string   `json:"privateKey"`
	Subnets    []string `json:"subnets"`
}

type proxy struct {
	ssm      []*ssmEntry
	ssh      []*sshEntry
	nextPort atomic.Int32
	ctx      context.Context
}

type ssmEntry struct {
	config   SSMConfig
	manager  *ssmSessionManager
	networks []*net.IPNet
}

type sshEntry struct {
	config    SSHConfig
	networks  []*net.IPNet
	sshClient *ssh.Client
	mu        sync.Mutex
}

type ssmSessionManager struct {
	client     *ssm.Client
	instanceID string
	region     string
	sessions   sync.Map
	nextPort   *atomic.Int32
	ctx        context.Context
}

type ssmSession struct {
	localPort int
	cmd       *exec.Cmd
	sessionID string
	cancel    context.CancelFunc
	lastUsed  time.Time
	mu        sync.Mutex
}

func newProxy(ctx context.Context, ssmConfigs []SSMConfig, sshConfigs []SSHConfig) (*proxy, error) {
	p := &proxy{
		ssm: make([]*ssmEntry, 0, len(ssmConfigs)),
		ssh: make([]*sshEntry, 0, len(sshConfigs)),
		ctx: ctx,
	}
	p.nextPort.Store(10000)

	for _, cfg := range ssmConfigs {
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config for region %s: %w", cfg.Region, err)
		}

		networks := parseCIDRs(cfg.Subnets)
		manager := &ssmSessionManager{
			client:     ssm.NewFromConfig(awsCfg),
			instanceID: cfg.InstanceID,
			region:     cfg.Region,
			nextPort:   &p.nextPort,
			ctx:        ctx,
		}

		p.ssm = append(p.ssm, &ssmEntry{
			config:   cfg,
			manager:  manager,
			networks: networks,
		})

		go manager.cleanupIdleSessions()
	}

	for _, cfg := range sshConfigs {
		p.ssh = append(p.ssh, &sshEntry{
			config:   cfg,
			networks: parseCIDRs(cfg.Subnets),
		})
	}

	return p, nil
}

func parseCIDRs(subnets []string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(subnets))
	for _, cidr := range subnets {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("failed to parse CIDR", "cidr", cidr, "error", err)
			continue
		}
		networks = append(networks, network)
	}
	return networks
}

func (p *proxy) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %w", addr, err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("failed to resolve hostname %s: %w", host, err)
		}
		ip = ips[0]
	}

	for _, entry := range p.ssm {
		for _, net := range entry.networks {
			if net.Contains(ip) {
				port, _ := strconv.Atoi(portStr)
				fmt.Println(ui.TEXT_INFO_BOLD.Render("| "), ui.TEXT_NORMAL.Render("Tunneling", network, addr, "via", entry.config.InstanceID))
				return entry.manager.dialThroughSSM(ctx, host, port)
			}
		}
	}

	for _, entry := range p.ssh {
		for _, net := range entry.networks {
			if net.Contains(ip) {
				fmt.Println(ui.TEXT_INFO_BOLD.Render("| "), ui.TEXT_NORMAL.Render("Tunneling", network, addr, "via SSH", entry.config.Host))
				sshClient, err := entry.getOrCreateSSHClient()
				if err != nil {
					return nil, err
				}
				return sshClient.Dial(network, addr)
			}
		}
	}

	return nil, fmt.Errorf("no tunnel found for IP %s", ip)
}

func (e *sshEntry) getOrCreateSSHClient() (*ssh.Client, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.sshClient != nil {
		_, _, err := e.sshClient.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			return e.sshClient, nil
		}
		e.sshClient.Close()
		e.sshClient = nil
	}

	signer, err := ssh.ParsePrivateKey([]byte(e.config.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: e.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	host := e.config.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "22")
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server %s: %w", host, err)
	}

	e.sshClient = client
	return client, nil
}

func (m *ssmSessionManager) cleanupIdleSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.sessions.Range(func(key, value interface{}) bool {
				if session, ok := value.(*ssmSession); ok {
					m.terminateSession(session)
				}
				return true
			})
			return
		case <-ticker.C:
			now := time.Now()
			m.sessions.Range(func(key, value interface{}) bool {
				if session, ok := value.(*ssmSession); ok {
					session.mu.Lock()
					if now.Sub(session.lastUsed) > 5*time.Minute {
						m.sessions.Delete(key)
						go m.terminateSession(session)
					}
					session.mu.Unlock()
				}
				return true
			})
		}
	}
}

func (m *ssmSessionManager) terminateSession(session *ssmSession) {
	if session.cancel != nil {
		session.cancel()
	}
	if session.cmd != nil && session.cmd.Process != nil {
		session.cmd.Process.Kill()
	}
	if session.sessionID != "" {
		m.client.TerminateSession(context.Background(), &ssm.TerminateSessionInput{
			SessionId: aws.String(session.sessionID),
		})
	}
}

func (m *ssmSessionManager) dialThroughSSM(ctx context.Context, remoteHost string, remotePort int) (net.Conn, error) {
	key := fmt.Sprintf("%s:%d", remoteHost, remotePort)

	if val, ok := m.sessions.Load(key); ok {
		session := val.(*ssmSession)
		session.mu.Lock()
		session.lastUsed = time.Now()
		session.mu.Unlock()

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", session.localPort))
		if err == nil {
			return conn, nil
		}
		m.sessions.Delete(key)
		m.terminateSession(session)
	}

	localPort := int(m.nextPort.Add(1))

	input := &ssm.StartSessionInput{
		Target:       aws.String(m.instanceID),
		DocumentName: aws.String("AWS-StartPortForwardingSessionToRemoteHost"),
		Parameters: map[string][]string{
			"host":            {remoteHost},
			"portNumber":      {strconv.Itoa(remotePort)},
			"localPortNumber": {strconv.Itoa(localPort)},
		},
	}

	output, err := m.client.StartSession(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to start SSM session: %w", err)
	}

	slog.Info("SSM session started", "sessionId", *output.SessionId, "remoteHost", remoteHost, "remotePort", remotePort, "localPort", localPort)

	sessionJSON, _ := json.Marshal(output)
	inputJSON, _ := json.Marshal(input)

	pluginCtx, cancel := context.WithCancel(m.ctx)
	pluginCmd := exec.CommandContext(pluginCtx, "session-manager-plugin",
		string(sessionJSON),
		m.region,
		"StartSession",
		"",
		string(inputJSON),
		fmt.Sprintf("https://ssm.%s.amazonaws.com", m.region),
	)
	pluginCmd.Stdout = io.Discard
	pluginCmd.Stderr = io.Discard

	if err := pluginCmd.Start(); err != nil {
		cancel()
		m.client.TerminateSession(context.Background(), &ssm.TerminateSessionInput{
			SessionId: output.SessionId,
		})
		return nil, fmt.Errorf("failed to start session-manager-plugin: %w", err)
	}

	session := &ssmSession{
		localPort: localPort,
		cmd:       pluginCmd,
		sessionID: *output.SessionId,
		cancel:    cancel,
		lastUsed:  time.Now(),
	}

	if err := m.waitForPort(ctx, localPort); err != nil {
		m.terminateSession(session)
		return nil, err
	}

	m.sessions.Store(key, session)

	go func() {
		pluginCmd.Wait()
		m.sessions.Delete(key)
	}()

	return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
}

func (m *ssmSessionManager) waitForPort(ctx context.Context, port int) error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for local port %d", port)
}

func StartProxy(ctx context.Context, ssmConfigs []SSMConfig, sshConfigs []SSHConfig) error {
	if len(ssmConfigs) == 0 && len(sshConfigs) == 0 {
		return fmt.Errorf("no tunnels configured")
	}

	p, err := newProxy(ctx, ssmConfigs, sshConfigs)
	if err != nil {
		return err
	}

	server, err := socks5.New(&socks5.Config{
		Dial: p.dial,
	})
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 server: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServe("tcp", "127.0.0.1:1080")
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}
