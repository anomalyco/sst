package cloudflare

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
)

func TestStaticSiteDeploymentBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN") == "" {
		t.Skip("Skipping Cloudflare integration test - SST_TEST_CLOUDFLARE_API_TOKEN not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("static-basic-%d", time.Now().Unix())
	
	// Create test project
	projectDir, err := helpers.CreateTestProject("cloudflare-static-basic", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-static-basic",
      region: "auto",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const site = new sst.cloudflare.StaticSite("TestSite", {
        path: "./public",
      });

      return {
        url: site.url,
      };
    });
  },
};`,
		"public/index.html": `<!DOCTYPE html>
<html>
<head>
    <title>Test Static Site</title>
</head>
<body>
    <h1>Hello from SST Static Site!</h1>
    <p>This is a test static site deployed to Cloudflare.</p>
</body>
</html>`,
		"public/about.html": `<!DOCTYPE html>
<html>
<head>
    <title>About - Test Static Site</title>
</head>
<body>
    <h1>About Page</h1>
    <p>This is the about page of our test static site.</p>
</body>
</html>`,
		"public/styles.css": `body {
    font-family: Arial, sans-serif;
    margin: 40px;
    background-color: #f5f5f5;
}

h1 {
    color: #333;
    border-bottom: 2px solid #007acc;
    padding-bottom: 10px;
}

p {
    line-height: 1.6;
    color: #666;
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to deploy static site project")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate outputs
	require.Contains(t, outputs, "url", "Stack should output site URL")
	siteURL := outputs["url"].(string)
	require.NotEmpty(t, siteURL, "Site URL should not be empty")
	require.True(t, strings.HasPrefix(siteURL, "https://"), "Site URL should use HTTPS")

	// Test site accessibility
	t.Run("Site is accessible", func(t *testing.T) {
		// Wait a bit for deployment to propagate
		time.Sleep(10 * time.Second)

		resp, err := http.Get(siteURL)
		require.NoError(t, err, "Failed to access site")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Site should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Hello from SST Static Site!", "Site should contain expected content")
		assert.Contains(t, bodyStr, "Test Static Site", "Site should contain page title")
	})

	// Test about page
	t.Run("About page is accessible", func(t *testing.T) {
		aboutURL := strings.TrimSuffix(siteURL, "/") + "/about.html"
		
		resp, err := http.Get(aboutURL)
		require.NoError(t, err, "Failed to access about page")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "About page should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read about page response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "About Page", "About page should contain expected content")
		assert.Contains(t, bodyStr, "about page of our test", "About page should contain description")
	})

	// Test CSS file
	t.Run("CSS file is accessible", func(t *testing.T) {
		cssURL := strings.TrimSuffix(siteURL, "/") + "/styles.css"
		
		resp, err := http.Get(cssURL)
		require.NoError(t, err, "Failed to access CSS file")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "CSS file should return 200 OK")
		assert.Equal(t, "text/css", resp.Header.Get("Content-Type"), "CSS file should have correct content type")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read CSS response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "font-family: Arial", "CSS should contain expected styles")
		assert.Contains(t, bodyStr, "background-color: #f5f5f5", "CSS should contain background color")
	})

	// Cleanup
	err = helpers.RemoveProject(ctx, projectDir, testStage)
	assert.NoError(t, err, "Failed to cleanup static site project")
}

func TestStaticSiteDeploymentWithBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN") == "" {
		t.Skip("Skipping Cloudflare integration test - SST_TEST_CLOUDFLARE_API_TOKEN not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("static-build-%d", time.Now().Unix())
	
	// Create test project with build process
	projectDir, err := helpers.CreateTestProject("cloudflare-static-build", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-static-build",
      region: "auto",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const site = new sst.cloudflare.StaticSite("TestSite", {
        build: {
          command: "npm run build",
          output: "dist"
        },
        environment: {
          VITE_APP_NAME: "SST Test App",
          VITE_BUILD_TIME: new Date().toISOString(),
        }
      });

      return {
        url: site.url,
      };
    });
  },
};`,
		"package.json": `{
  "name": "test-static-build",
  "version": "1.0.0",
  "scripts": {
    "build": "mkdir -p dist && cp -r src/* dist/",
    "dev": "echo 'Dev mode not implemented'"
  },
  "devDependencies": {}
}`,
		"src/index.html": `<!DOCTYPE html>
<html>
<head>
    <title>Built Static Site</title>
    <meta name="description" content="A static site built with SST">
</head>
<body>
    <h1>Built with SST!</h1>
    <p>This site was built using a build process.</p>
    <div id="build-info">
        <p>App: <span id="app-name">Loading...</span></p>
        <p>Built at: <span id="build-time">Loading...</span></p>
    </div>
    <script>
        // Simulate environment variable usage
        document.getElementById('app-name').textContent = 'SST Test App';
        document.getElementById('build-time').textContent = new Date().toISOString();
    </script>
</body>
</html>`,
		"src/robots.txt": `User-agent: *
Allow: /

Sitemap: /sitemap.xml`,
		"src/sitemap.xml": `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <lastmod>2024-01-01</lastmod>
    <priority>1.0</priority>
  </url>
</urlset>`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to deploy static site with build project")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate outputs
	require.Contains(t, outputs, "url", "Stack should output site URL")
	siteURL := outputs["url"].(string)
	require.NotEmpty(t, siteURL, "Site URL should not be empty")

	// Test built site accessibility
	t.Run("Built site is accessible", func(t *testing.T) {
		// Wait for deployment to propagate
		time.Sleep(10 * time.Second)

		resp, err := http.Get(siteURL)
		require.NoError(t, err, "Failed to access built site")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Built site should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Built with SST!", "Site should contain expected content")
		assert.Contains(t, bodyStr, "built using a build process", "Site should contain build description")
		assert.Contains(t, bodyStr, "build-info", "Site should contain build info section")
	})

	// Test robots.txt
	t.Run("Robots.txt is accessible", func(t *testing.T) {
		robotsURL := strings.TrimSuffix(siteURL, "/") + "/robots.txt"
		
		resp, err := http.Get(robotsURL)
		require.NoError(t, err, "Failed to access robots.txt")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Robots.txt should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read robots.txt response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "User-agent: *", "Robots.txt should contain user agent directive")
		assert.Contains(t, bodyStr, "Allow: /", "Robots.txt should allow all paths")
	})

	// Test sitemap.xml
	t.Run("Sitemap.xml is accessible", func(t *testing.T) {
		sitemapURL := strings.TrimSuffix(siteURL, "/") + "/sitemap.xml"
		
		resp, err := http.Get(sitemapURL)
		require.NoError(t, err, "Failed to access sitemap.xml")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Sitemap.xml should return 200 OK")
		assert.Contains(t, resp.Header.Get("Content-Type"), "xml", "Sitemap should have XML content type")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read sitemap.xml response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "<?xml version", "Sitemap should be valid XML")
		assert.Contains(t, bodyStr, "<urlset", "Sitemap should contain urlset")
		assert.Contains(t, bodyStr, "<loc>", "Sitemap should contain location entries")
	})

	// Cleanup
	err = helpers.RemoveProject(ctx, projectDir, testStage)
	assert.NoError(t, err, "Failed to cleanup static site with build project")
}

func TestStaticSiteDeploymentWithCustomDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN") == "" {
		t.Skip("Skipping Cloudflare integration test - SST_TEST_CLOUDFLARE_API_TOKEN not set")
	}

	if os.Getenv("SST_TEST_CLOUDFLARE_ZONE_ID") == "" {
		t.Skip("Skipping Cloudflare custom domain test - SST_TEST_CLOUDFLARE_ZONE_ID not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("static-domain-%d", time.Now().Unix())
	
	// Create test project with custom domain
	projectDir, err := helpers.CreateTestProject("cloudflare-static-domain", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-static-domain",
      region: "auto",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const site = new sst.cloudflare.StaticSite("TestSite", {
        path: "./public",
        domain: {
          name: "test-static-" + Date.now() + ".example.com",
          dns: sst.cloudflare.dns(),
        },
      });

      return {
        url: site.url,
        domain: site.domain,
      };
    });
  },
};`,
		"public/index.html": `<!DOCTYPE html>
<html>
<head>
    <title>Custom Domain Static Site</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
    <h1>Custom Domain Site</h1>
    <p>This static site is served from a custom domain.</p>
    <div class="info">
        <h2>Domain Features</h2>
        <ul>
            <li>HTTPS enabled by default</li>
            <li>Cloudflare CDN acceleration</li>
            <li>Global edge distribution</li>
        </ul>
    </div>
</body>
</html>`,
		"public/health.json": `{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "service": "static-site",
  "version": "1.0.0"
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to deploy static site with custom domain")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate outputs
	require.Contains(t, outputs, "url", "Stack should output site URL")
	require.Contains(t, outputs, "domain", "Stack should output custom domain")
	
	siteURL := outputs["url"].(string)
	customDomain := outputs["domain"].(string)
	
	require.NotEmpty(t, siteURL, "Site URL should not be empty")
	require.NotEmpty(t, customDomain, "Custom domain should not be empty")
	require.True(t, strings.HasPrefix(siteURL, "https://"), "Site URL should use HTTPS")

	// Test custom domain accessibility
	t.Run("Custom domain is accessible", func(t *testing.T) {
		// Wait longer for DNS propagation
		time.Sleep(30 * time.Second)

		resp, err := http.Get(siteURL)
		require.NoError(t, err, "Failed to access site via custom domain")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Custom domain site should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Custom Domain Site", "Site should contain expected content")
		assert.Contains(t, bodyStr, "custom domain", "Site should mention custom domain")
		assert.Contains(t, bodyStr, "HTTPS enabled", "Site should mention HTTPS")
	})

	// Test health endpoint
	t.Run("Health endpoint returns JSON", func(t *testing.T) {
		healthURL := strings.TrimSuffix(siteURL, "/") + "/health.json"
		
		resp, err := http.Get(healthURL)
		require.NoError(t, err, "Failed to access health endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200 OK")
		assert.Contains(t, resp.Header.Get("Content-Type"), "json", "Health endpoint should return JSON")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read health response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, `"status": "healthy"`, "Health endpoint should return healthy status")
		assert.Contains(t, bodyStr, `"service": "static-site"`, "Health endpoint should identify service")
	})

	// Test HTTPS security headers
	t.Run("HTTPS security headers are present", func(t *testing.T) {
		resp, err := http.Get(siteURL)
		require.NoError(t, err, "Failed to access site for security header check")
		defer resp.Body.Close()

		// Check for common security headers that Cloudflare might add
		headers := resp.Header
		
		// These headers might be present depending on Cloudflare configuration
		if headers.Get("Strict-Transport-Security") != "" {
			assert.Contains(t, headers.Get("Strict-Transport-Security"), "max-age", "HSTS header should contain max-age")
		}
		
		if headers.Get("X-Content-Type-Options") != "" {
			assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"), "X-Content-Type-Options should be nosniff")
		}
		
		// Cloudflare should add some identifying headers
		assert.NotEmpty(t, headers.Get("Cf-Ray"), "Cloudflare Ray ID should be present")
	})

	// Cleanup
	err = helpers.RemoveProject(ctx, projectDir, testStage)
	assert.NoError(t, err, "Failed to cleanup static site with custom domain")
}

func TestStaticSiteDeploymentWithAssets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN") == "" {
		t.Skip("Skipping Cloudflare integration test - SST_TEST_CLOUDFLARE_API_TOKEN not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("static-assets-%d", time.Now().Unix())
	
	// Create test project with various asset types
	projectDir, err := helpers.CreateTestProject("cloudflare-static-assets", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-static-assets",
      region: "auto",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const site = new sst.cloudflare.StaticSite("TestSite", {
        path: "./public",
        assets: {
          "*.css": {
            cacheControl: "public, max-age=31536000, immutable"
          },
          "*.js": {
            cacheControl: "public, max-age=31536000, immutable"
          },
          "*.png": {
            cacheControl: "public, max-age=2592000"
          },
          "*.html": {
            cacheControl: "public, max-age=0, must-revalidate"
          }
        }
      });

      return {
        url: site.url,
      };
    });
  },
};`,
		"public/index.html": `<!DOCTYPE html>
<html>
<head>
    <title>Asset Test Site</title>
    <link rel="stylesheet" href="/styles.css">
    <link rel="icon" href="/favicon.png">
</head>
<body>
    <h1>Asset Testing</h1>
    <p>This site tests various asset types and caching.</p>
    <img src="/test-image.png" alt="Test Image" width="100" height="100">
    <script src="/app.js"></script>
</body>
</html>`,
		"public/styles.css": `body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    margin: 0;
    padding: 20px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    min-height: 100vh;
}

h1 {
    text-align: center;
    margin-bottom: 30px;
    text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
}

p {
    text-align: center;
    font-size: 18px;
    margin-bottom: 20px;
}

img {
    display: block;
    margin: 20px auto;
    border-radius: 10px;
    box-shadow: 0 4px 8px rgba(0,0,0,0.3);
}`,
		"public/app.js": `console.log('SST Static Site Assets Test');

document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded');
    
    // Add some interactive functionality
    const heading = document.querySelector('h1');
    if (heading) {
        heading.addEventListener('click', function() {
            this.style.transform = this.style.transform === 'scale(1.1)' ? 'scale(1)' : 'scale(1.1)';
            this.style.transition = 'transform 0.3s ease';
        });
    }
    
    // Log asset loading
    const img = document.querySelector('img');
    if (img) {
        img.onload = function() {
            console.log('Image loaded successfully');
        };
        img.onerror = function() {
            console.error('Failed to load image');
        };
    }
});

// Test function for validation
function getAssetInfo() {
    return {
        timestamp: new Date().toISOString(),
        userAgent: navigator.userAgent,
        location: window.location.href
    };
}`,
		// Create a simple PNG image as base64 (1x1 transparent pixel)
		"public/test-image.png": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
		"public/favicon.png": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to deploy static site with assets")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate outputs
	require.Contains(t, outputs, "url", "Stack should output site URL")
	siteURL := outputs["url"].(string)
	require.NotEmpty(t, siteURL, "Site URL should not be empty")

	// Test main page with assets
	t.Run("Main page loads with assets", func(t *testing.T) {
		// Wait for deployment to propagate
		time.Sleep(10 * time.Second)

		resp, err := http.Get(siteURL)
		require.NoError(t, err, "Failed to access site")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Site should return 200 OK")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Asset Testing", "Site should contain expected content")
		assert.Contains(t, bodyStr, "styles.css", "Site should reference CSS file")
		assert.Contains(t, bodyStr, "app.js", "Site should reference JS file")
		assert.Contains(t, bodyStr, "test-image.png", "Site should reference image file")
	})

	// Test CSS file with caching headers
	t.Run("CSS file has correct caching headers", func(t *testing.T) {
		cssURL := strings.TrimSuffix(siteURL, "/") + "/styles.css"
		
		resp, err := http.Get(cssURL)
		require.NoError(t, err, "Failed to access CSS file")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "CSS file should return 200 OK")
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/css", "CSS file should have correct content type")

		// Check for cache control headers (may be set by Cloudflare)
		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "" {
			assert.Contains(t, cacheControl, "public", "CSS should have public cache control")
		}

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read CSS response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "font-family", "CSS should contain font styles")
		assert.Contains(t, bodyStr, "background: linear-gradient", "CSS should contain gradient background")
	})

	// Test JavaScript file
	t.Run("JavaScript file is accessible", func(t *testing.T) {
		jsURL := strings.TrimSuffix(siteURL, "/") + "/app.js"
		
		resp, err := http.Get(jsURL)
		require.NoError(t, err, "Failed to access JS file")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "JS file should return 200 OK")
		assert.Contains(t, resp.Header.Get("Content-Type"), "javascript", "JS file should have correct content type")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read JS response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "SST Static Site Assets Test", "JS should contain expected content")
		assert.Contains(t, bodyStr, "DOMContentLoaded", "JS should contain DOM ready handler")
		assert.Contains(t, bodyStr, "getAssetInfo", "JS should contain test function")
	})

	// Test image file
	t.Run("Image file is accessible", func(t *testing.T) {
		imgURL := strings.TrimSuffix(siteURL, "/") + "/test-image.png"
		
		resp, err := http.Get(imgURL)
		require.NoError(t, err, "Failed to access image file")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Image file should return 200 OK")
		assert.Contains(t, resp.Header.Get("Content-Type"), "image", "Image file should have correct content type")

		// Check that we got some content (even if it's a data URL)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read image response body")
		assert.NotEmpty(t, body, "Image response should not be empty")
	})

	// Cleanup
	err = helpers.RemoveProject(ctx, projectDir, testStage)
	assert.NoError(t, err, "Failed to cleanup static site with assets")
}