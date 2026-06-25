<?php

namespace Sst\Sdk\Tests;

use PHPUnit\Framework\TestCase;
use Sst\Sdk\Resource;

class ResourceTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();
        // Reset the Resource class state between tests
        $reflection = new \ReflectionClass(Resource::class);
        $property = $reflection->getProperty('resources');
        $property->setAccessible(true);
        $property->setValue(null, null);
    }

    protected function tearDown(): void
    {
        parent::tearDown();
        // Clean up environment variables
        putenv('SST_RESOURCE_App');
        putenv('SST_RESOURCE_MyBucket');
        putenv('SST_RESOURCE_MyTable');
        putenv('SST_KEY');
        putenv('SST_KEY_FILE');
        putenv('AWS_LAMBDA_FUNCTION_NAME');
        
        // Also clean $_ENV
        unset($_ENV['SST_RESOURCE_App']);
        unset($_ENV['SST_RESOURCE_MyBucket']);
        unset($_ENV['SST_RESOURCE_MyTable']);
        unset($_ENV['SST_KEY']);
        unset($_ENV['SST_KEY_FILE']);
        unset($_ENV['AWS_LAMBDA_FUNCTION_NAME']);
    }

    public function testGetResourceFromEnvironmentVariable(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket', 'type' => 'sst.aws.Bucket']);

        $app = Resource::get('App');
        $this->assertEquals(['name' => 'my-app', 'stage' => 'dev'], $app);

        $bucket = Resource::get('MyBucket');
        $this->assertEquals(['name' => 'my-bucket', 'type' => 'sst.aws.Bucket'], $bucket);
    }

    public function testGetResourceFromGetenv(): void
    {
        putenv('SST_RESOURCE_App=' . json_encode(['name' => 'test-app', 'stage' => 'prod']));
        putenv('SST_RESOURCE_MyTable=' . json_encode(['name' => 'test-table', 'type' => 'sst.aws.Dynamo']));

        $app = Resource::get('App');
        $this->assertEquals(['name' => 'test-app', 'stage' => 'prod'], $app);

        $table = Resource::get('MyTable');
        $this->assertEquals(['name' => 'test-table', 'type' => 'sst.aws.Dynamo'], $table);
    }

    public function testThrowsErrorWhenResourceNotFound(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);

        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage('"NonExistentResource" is not linked in your sst.config.ts');

        Resource::get('NonExistentResource');
    }

    public function testThrowsErrorWithLambdaFunctionName(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        putenv('AWS_LAMBDA_FUNCTION_NAME=my-function');

        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage('"MissingResource" is not linked in your sst.config.ts to my-function');

        Resource::get('MissingResource');
    }

    public function testThrowsErrorWhenSSTLinksNotActive(): void
    {
        // Don't set SST_RESOURCE_App
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'bucket']);

        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage('It does not look like SST links are active');

        Resource::get('SomeResource');
    }

    public function testDecryptFromKeyFile(): void
    {
        // Create a temporary encrypted file for testing
        // This uses the same encryption as the other SDKs
        $key = random_bytes(32);
        $nonce = str_repeat("\0", 12);
        $data = json_encode([
            'App' => ['name' => 'encrypted-app', 'stage' => 'test'],
            'SecretBucket' => ['name' => 'secret-bucket', 'encrypted' => true]
        ]);

        $ciphertext = openssl_encrypt(
            $data,
            'aes-256-gcm',
            $key,
            OPENSSL_RAW_DATA,
            $nonce,
            $tag
        );

        $encryptedData = $ciphertext . $tag;

        // Write to temporary file
        $tempFile = tempnam(sys_get_temp_dir(), 'sst_test_');
        file_put_contents($tempFile, $encryptedData);

        // Set environment variables
        putenv('SST_KEY=' . base64_encode($key));
        putenv('SST_KEY_FILE=' . $tempFile);

        try {
            $app = Resource::get('App');
            $this->assertEquals(['name' => 'encrypted-app', 'stage' => 'test'], $app);

            $bucket = Resource::get('SecretBucket');
            $this->assertEquals(['name' => 'secret-bucket', 'encrypted' => true], $bucket);
        } finally {
            // Clean up
            unlink($tempFile);
        }
    }

    public function testEncryptedFileWithEnvironmentVariableOverride(): void
    {
        // Create encrypted file with one resource
        $key = random_bytes(32);
        $nonce = str_repeat("\0", 12);
        $data = json_encode([
            'App' => ['name' => 'encrypted-app', 'stage' => 'test'],
            'EncryptedResource' => ['source' => 'encrypted']
        ]);

        $ciphertext = openssl_encrypt(
            $data,
            'aes-256-gcm',
            $key,
            OPENSSL_RAW_DATA,
            $nonce,
            $tag
        );

        $encryptedData = $ciphertext . $tag;
        $tempFile = tempnam(sys_get_temp_dir(), 'sst_test_');
        file_put_contents($tempFile, $encryptedData);

        putenv('SST_KEY=' . base64_encode($key));
        putenv('SST_KEY_FILE=' . $tempFile);
        
        // Add an environment variable resource
        $_ENV['SST_RESOURCE_EnvResource'] = json_encode(['source' => 'environment']);

        try {
            $encrypted = Resource::get('EncryptedResource');
            $this->assertEquals(['source' => 'encrypted'], $encrypted);

            $env = Resource::get('EnvResource');
            $this->assertEquals(['source' => 'environment'], $env);
        } finally {
            unlink($tempFile);
        }
    }

    public function testFallbackToEnvWhenDecryptionFails(): void
    {
        // Set invalid key file
        $tempFile = tempnam(sys_get_temp_dir(), 'sst_test_');
        file_put_contents($tempFile, 'invalid encrypted data');

        putenv('SST_KEY=' . base64_encode(random_bytes(32)));
        putenv('SST_KEY_FILE=' . $tempFile);
        
        // Should fall back to environment variables
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'fallback-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'env-bucket']);

        try {
            $app = Resource::get('App');
            $this->assertEquals(['name' => 'fallback-app', 'stage' => 'dev'], $app);

            $bucket = Resource::get('MyBucket');
            $this->assertEquals(['name' => 'env-bucket'], $bucket);
        } finally {
            unlink($tempFile);
        }
    }

    public function testLazyInitialization(): void
    {
        // Verify resources are only loaded on first access
        $reflection = new \ReflectionClass(Resource::class);
        $property = $reflection->getProperty('resources');
        $property->setAccessible(true);

        // Before any access, resources should be null
        $this->assertNull($property->getValue());

        // Set an environment variable
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'test', 'stage' => 'dev']);

        // After first access, resources should be initialized
        Resource::get('App');
        $this->assertIsArray($property->getValue());
    }

    public function testInvalidJsonInEnvironmentVariableIsSkipped(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_InvalidJson'] = 'not-valid-json{';
        $_ENV['SST_RESOURCE_ValidResource'] = json_encode(['valid' => true]);

        // Should not throw, just skip the invalid JSON
        $app = Resource::get('App');
        $this->assertEquals(['name' => 'my-app', 'stage' => 'dev'], $app);

        $valid = Resource::get('ValidResource');
        $this->assertEquals(['valid' => true], $valid);

        // InvalidJson should not be accessible
        $this->expectException(\RuntimeException::class);
        Resource::get('InvalidJson');
    }

    public function testEmptyEnvironmentVariablesAreHandled(): void
    {
        putenv('SST_KEY=');
        putenv('SST_KEY_FILE=');
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'test', 'stage' => 'dev']);

        // Should still work with just environment variables
        $app = Resource::get('App');
        $this->assertEquals(['name' => 'test', 'stage' => 'dev'], $app);
    }

    public function testGetResourceWithPath(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket', 'type' => 'sst.aws.Bucket']);

        // Get nested property directly
        $bucketName = Resource::get('MyBucket', 'name');
        $this->assertEquals('my-bucket', $bucketName);

        $bucketType = Resource::get('MyBucket', 'type');
        $this->assertEquals('sst.aws.Bucket', $bucketType);
    }

    public function testGetResourceWithNestedPath(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyResource'] = json_encode([
            'config' => [
                'nested' => [
                    'value' => 'deep-value'
                ]
            ]
        ]);

        // Get deeply nested property
        $value = Resource::get('MyResource', 'config', 'nested', 'value');
        $this->assertEquals('deep-value', $value);
    }

    public function testGetResourceWithInvalidPath(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket']);

        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage('Resource path not found');

        Resource::get('MyBucket', 'nonexistent');
    }

    public function testGetResourceWithPathOnNonArray(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket']);

        $this->expectException(\RuntimeException::class);
        $this->expectExceptionMessage('Resource path not found');

        // Try to traverse a string value
        Resource::get('MyBucket', 'name', 'invalid');
    }

    public function testAllMethod(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket']);
        $_ENV['SST_RESOURCE_MyTable'] = json_encode(['name' => 'my-table']);

        $all = Resource::all();

        $this->assertIsArray($all);
        $this->assertArrayHasKey('App', $all);
        $this->assertArrayHasKey('MyBucket', $all);
        $this->assertArrayHasKey('MyTable', $all);
        $this->assertEquals(['name' => 'my-app', 'stage' => 'dev'], $all['App']);
        $this->assertEquals(['name' => 'my-bucket'], $all['MyBucket']);
        $this->assertEquals(['name' => 'my-table'], $all['MyTable']);
    }

    public function testGetAndPathWorkTogether(): void
    {
        $_ENV['SST_RESOURCE_App'] = json_encode(['name' => 'my-app', 'stage' => 'dev']);
        $_ENV['SST_RESOURCE_MyBucket'] = json_encode(['name' => 'my-bucket', 'region' => 'us-east-1']);

        // Get whole resource
        $bucket = Resource::get('MyBucket');
        $this->assertEquals(['name' => 'my-bucket', 'region' => 'us-east-1'], $bucket);

        // Get specific property
        $name = Resource::get('MyBucket', 'name');
        $this->assertEquals('my-bucket', $name);

        // Both should work
        $this->assertEquals($bucket['name'], $name);
    }
}

