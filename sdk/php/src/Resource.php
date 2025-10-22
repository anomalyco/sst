<?php

namespace Sst\Sdk;

class Resource {
    private static ?array $resources = null;

    private static function init(): void {
        self::$resources = [];

        // Try to load from encrypted file if SST_KEY_FILE and SST_KEY are present
        $sstKey = getenv('SST_KEY');
        $sstKeyFile = getenv('SST_KEY_FILE');

        if ($sstKey !== false && $sstKeyFile !== false && $sstKey !== '' && $sstKeyFile !== '') {
            try {
                // Decode the base64-encoded key
                $key = base64_decode($sstKey, true);
                if ($key === false) {
                    throw new \RuntimeException('Failed to decode SST_KEY from base64');
                }

                // Read the encrypted data from the file
                $encryptedData = @file_get_contents($sstKeyFile);
                if ($encryptedData === false) {
                    throw new \RuntimeException("Failed to read SST_KEY_FILE: $sstKeyFile");
                }

                // Extract auth tag (last 16 bytes) and ciphertext
                $authTagStart = strlen($encryptedData) - 16;
                $actualCiphertext = substr($encryptedData, 0, $authTagStart);
                $authTag = substr($encryptedData, $authTagStart);

                // Create a 12-byte zero nonce
                $nonce = str_repeat("\0", 12);

                // Decrypt using AES-256-GCM
                $decrypted = openssl_decrypt(
                    $actualCiphertext,
                    'aes-256-gcm',
                    $key,
                    OPENSSL_RAW_DATA,
                    $nonce,
                    $authTag
                );

                if ($decrypted === false) {
                    throw new \RuntimeException('Failed to decrypt SST_KEY_FILE');
                }

                // Parse the decrypted JSON data
                $decryptedData = json_decode($decrypted, true);
                if ($decryptedData === null && json_last_error() !== JSON_ERROR_NONE) {
                    throw new \RuntimeException('Failed to parse decrypted data as JSON: ' . json_last_error_msg());
                }

                // Merge decrypted data into resources
                self::$resources = array_merge(self::$resources, $decryptedData);
            } catch (\Exception $e) {
                // If decryption fails, continue with environment variables only
                // This allows fallback to env-only mode
            }
        }

        // Load resources from environment variables with SST_RESOURCE_ prefix
        foreach ($_ENV as $key => $value) {
            if (strpos($key, 'SST_RESOURCE_') === 0) {
                $resourceName = substr($key, strlen('SST_RESOURCE_'));
                $decoded = json_decode($value, true);
                if ($decoded === null && json_last_error() !== JSON_ERROR_NONE) {
                    // If JSON decode fails, skip this resource
                    continue;
                }
                self::$resources[$resourceName] = $decoded;
            }
        }

        // Also check getenv() for environments where $_ENV might not be populated
        $env = getenv();
        if (is_array($env)) {
            foreach ($env as $key => $value) {
                if (strpos($key, 'SST_RESOURCE_') === 0) {
                    $resourceName = substr($key, strlen('SST_RESOURCE_'));
                    // Don't override if already set from $_ENV
                    if (!isset(self::$resources[$resourceName])) {
                        $decoded = json_decode($value, true);
                        if ($decoded === null && json_last_error() !== JSON_ERROR_NONE) {
                            continue;
                        }
                        self::$resources[$resourceName] = $decoded;
                    }
                }
            }
        }
    }

    public static function get(string $name, string ...$path) {
        // Lazy initialization
        if (self::$resources === null) {
            self::init();
        }

        // Check if the resource exists
        if (!isset(self::$resources[$name])) {
            // If resource not found, provide helpful error messages
            if (!isset(self::$resources['App'])) {
                throw new \RuntimeException(
                    'It does not look like SST links are active. If this is in local development and you are not starting this process through the multiplexer, wrap your command with `sst dev -- <command>`'
                );
            }

            $msg = "\"$name\" is not linked in your sst.config.ts";
            if (getenv('AWS_LAMBDA_FUNCTION_NAME') !== false) {
                $msg .= ' to ' . getenv('AWS_LAMBDA_FUNCTION_NAME');
            }

            throw new \RuntimeException($msg);
        }

        // If no path specified, return the whole resource
        if (empty($path)) {
            return self::$resources[$name];
        }

        // Traverse the path
        return self::getByPath(self::$resources[$name], $path);
    }

    public static function all(): array {
        // Lazy initialization
        if (self::$resources === null) {
            self::init();
        }

        return self::$resources;
    }

    private static function getByPath($input, array $path) {
        if (empty($path)) {
            return $input;
        }

        if (!is_array($input)) {
            throw new \RuntimeException('Resource path not found');
        }

        $key = array_shift($path);
        if (!isset($input[$key])) {
            throw new \RuntimeException('Resource path not found');
        }

        return self::getByPath($input[$key], $path);
    }
}
