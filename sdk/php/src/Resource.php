<?php

namespace Sst\Sdk;

class Resource {
    private static ?array $resources = null;

    private static function init(): void {
        self::$resources = [];

        $sstKey = getenv('SST_KEY');
        $sstKeyFile = getenv('SST_KEY_FILE');

        if (!empty($sstKey) && !empty($sstKeyFile)) {
            try {
                $key = base64_decode($sstKey, true);
                if ($key === false) {
                    throw new \RuntimeException('Failed to decode SST_KEY from base64');
                }

                $encryptedData = @file_get_contents($sstKeyFile);
                if ($encryptedData === false) {
                    throw new \RuntimeException("Failed to read SST_KEY_FILE: $sstKeyFile");
                }

                $authTagStart = strlen($encryptedData) - 16;
                $actualCiphertext = substr($encryptedData, 0, $authTagStart);
                $authTag = substr($encryptedData, $authTagStart);

                $nonce = str_repeat("\0", 12);

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

                $decryptedData = json_decode($decrypted, true);
                if ($decryptedData === null && json_last_error() !== JSON_ERROR_NONE) {
                    throw new \RuntimeException('Failed to parse decrypted data as JSON: ' . json_last_error_msg());
                }

                self::$resources = array_merge(self::$resources, $decryptedData);
            } catch (\Exception $e) {
            }
        }

        self::loadResourcesFromEnv($_ENV);
        
        $env = getenv();
        if (is_array($env)) {
            self::loadResourcesFromEnv($env);
        }
    }

    private static function loadResourcesFromEnv(array $env): void {
        foreach ($env as $key => $value) {
            if (strpos($key, 'SST_RESOURCE_') === 0) {
                $resourceName = substr($key, strlen('SST_RESOURCE_'));
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

    public static function get(string $name, string ...$path) {
        if (self::$resources === null) {
            self::init();
        }

        if (!isset(self::$resources[$name])) {
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

        if (empty($path)) {
            return self::$resources[$name];
        }

        return self::getByPath(self::$resources[$name], $path);
    }

    public static function all(): array {
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
