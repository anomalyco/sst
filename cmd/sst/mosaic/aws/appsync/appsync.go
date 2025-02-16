package appsync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/gorilla/websocket"
	"github.com/sst/sst/v3/pkg/id"
)

var log = slog.Default().With("service", "appsync.connection")

var ErrSubscriptionFailed = fmt.Errorf("appsync subscription failed")
var ErrConnectionFailed = fmt.Errorf("appsync connection failed")

type Connection struct {
	conn             *websocket.Conn
	cfg              aws.Config
	httpEndpoint     string
	realtimeEndpoint string
	subscriptions    map[string]SubscriptionInfo
	lock             sync.Mutex
}

type SubscriptionInfo struct {
	Channel string
	Out     chan string
}

type SubscribeEvent struct {
	Type          string      `json:"type"`
	ID            string      `json:"id"`
	Channel       string      `json:"channel"`
	Authorization interface{} `json:"authorization"`
}

func Dial(
	ctx context.Context,
	cfg aws.Config,
	httpEndpoint string,
	realtimeEndpoint string,
) (*Connection, error) {
	result := &Connection{
		cfg:              cfg,
		httpEndpoint:     httpEndpoint,
		realtimeEndpoint: realtimeEndpoint,
		subscriptions:    map[string]SubscriptionInfo{},
	}

	err := result.connect(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		if result.conn != nil {
			result.conn.Close()
		}
		for _, item := range result.subscriptions {
			close(item.Out)
		}
	}()
	return result, nil
}

func (c *Connection) connect(ctx context.Context) error {
	log.Info("connecting")
	auth, err := c.getAuth(ctx, map[string]interface{}{})
	if err != nil {
		return err
	}
	authJson, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	auth64 := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(authJson)
	dialer := websocket.Dialer{
		Subprotocols: []string{"aws-appsync-event-ws", "header-" + auth64},
	}
	conn, _, err := dialer.DialContext(ctx, "wss://"+c.realtimeEndpoint+"/event/realtime", nil)
	if err != nil {
		return err
	}
	conn.WriteJSON(map[string]interface{}{
		"type": "connection_init",
	})
	c.conn = conn

	msg := map[string]interface{}{}
	err = conn.ReadJSON(&msg)
	if err != nil {
		log.Error("write to connection failed", "err", err)
		return ErrConnectionFailed
	}
	log.Info("connect message", "msg", msg)
	if msg["type"] != "connection_ack" {
		return ErrConnectionFailed
	}
	duration := time.Millisecond * time.Duration(msg["connectionTimeoutMs"].(float64))

	timer := time.NewTimer(duration)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			log.Info("connection timeout")
			conn.Close()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := c.connect(ctx)
					if err != nil {
						log.Info("failed to reconnect", "err", err)
						time.Sleep(time.Second * 5)
						continue
					}
					for id, sub := range c.subscriptions {
						log.Info("resubscribing", "channel", sub.Channel, "id", id)
						err := c.subscribe(ctx, sub.Channel, id)
						if err != nil {
							log.Error("failed to resubscribe", "err", err)
							continue
						}
					}
					return
				}
			}
		}
	}()

	go func() {
		for {
			msg := map[string]interface{}{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Info("connection closed")
				timer.Reset(1 * time.Millisecond)
				return
			}
			log.Info("msg", "type", msg["type"], "id", msg["id"])

			if msg["type"] == "connection_ack" {
				duration = time.Millisecond * time.Duration(msg["connectionTimeoutMs"].(float64))
				log.Info("keep alive set", "duration", duration.Seconds())
				timer.Reset(duration)
			}

			if msg["type"] == "ka" {
				timer.Reset(duration)
			}

			if msg["type"] == "subscribe_success" {
				id := msg["id"].(string)
				if item, ok := c.subscriptions[id]; ok {
					item.Out <- "ok"
				}
			}
			if t := msg["type"]; t == "data" {
				id := msg["id"].(string)
				if item, ok := c.subscriptions[id]; ok {
					item.Out <- msg["event"].(string)
				}
			}
		}
	}()

	return nil
}

func (c *Connection) Subscribe(ctx context.Context, channel string) (chan string, error) {
	out := make(chan string, 1000)
	subscriptionID := id.Ascending()
	c.subscriptions[subscriptionID] = SubscriptionInfo{
		Channel: channel,
		Out:     out,
	}
	err := c.subscribe(ctx, channel, subscriptionID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Connection) subscribe(ctx context.Context, channel string, subscriptionID string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	auth, err := c.getAuth(ctx, map[string]interface{}{
		"channel": channel,
	})
	if err != nil {
		return err
	}
	old := c.subscriptions[subscriptionID].Out
	tmp := make(chan string, 1)
	defer func() {
		c.subscriptions[subscriptionID] = SubscriptionInfo{
			Channel: channel,
			Out:     old,
		}
	}()
	c.subscriptions[subscriptionID] = SubscriptionInfo{
		Channel: channel,
		Out:     tmp,
	}
	c.conn.WriteJSON(map[string]interface{}{
		"type":          "subscribe",
		"id":            subscriptionID,
		"channel":       channel,
		"authorization": auth,
	})
	select {
	case <-tmp:
		log.Info("subscribed", "channel", channel, "id", subscriptionID)
		return nil
	case <-time.After(time.Second * 3):
		return ErrSubscriptionFailed
	}
}

func (c *Connection) getAuth(ctx context.Context, body interface{}) (interface{}, error) {
	credentials, err := c.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader := bytes.NewReader(bodyJson)
	req, err := http.NewRequest("POST",
		"https://"+c.httpEndpoint+"/event",
		bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json, text/javascript")
	req.Header.Set("content-encoding", "amz-1.0")
	req.Header.Set("content-type", "application/json; charset=UTF-8")

	// Compute SHA256 hash of the payload
	h := sha256.New()
	h.Write(bodyJson)
	payloadHash := hex.EncodeToString(h.Sum(nil))

	signer := v4.NewSigner()
	err = signer.SignHTTP(ctx,
		credentials,
		req,
		payloadHash,
		"appsync",
		c.cfg.Region,
		time.Now(),
	)
	auth := map[string]string{
		"accept":           req.Header.Get("accept"),
		"content-encoding": req.Header.Get("content-encoding"),
		"content-type":     req.Header.Get("content-type"),
		"host":             req.Host,
		"x-amz-date":       req.Header.Get("x-amz-date"),
		"Authorization":    req.Header.Get("Authorization"),
	}
	if req.Header.Get("X-Amz-Security-Token") != "" {
		auth["X-Amz-Security-Token"] = req.Header.Get("X-Amz-Security-Token")
	}
	return auth, nil
}

func (c *Connection) Publish(ctx context.Context, channel string, event interface{}) error {
	credentials, err := c.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return err
	}
	eventJson, err := json.Marshal(event)
	body, err := json.Marshal(map[string]interface{}{
		"channel": channel,
		"events":  []string{string(eventJson)},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "https://"+c.httpEndpoint+"/event", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	signer := v4.NewSigner()
	h := sha256.New()
	h.Write(body)
	payloadHash := hex.EncodeToString(h.Sum(nil))
	err = signer.SignHTTP(ctx, credentials, req, payloadHash, "appsync", c.cfg.Region, time.Now())
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
