use aes_gcm::{
    aead::{generic_array::GenericArray, Aead, KeyInit},
    Aes256Gcm,
};
use base64::prelude::*;
use serde::de::DeserializeOwned;
use serde_json::Value;
use std::{collections::HashMap, env};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ResourceError {
    #[error("Resource not found")]
    NotFound,
    #[error("Environment error: {0}")]
    EnvError(#[from] std::env::VarError),
    #[error("IO error: {0}")]
    IoError(#[from] std::io::Error),
    #[error("Decryption error: {0}")]
    DecryptionError(String),
    #[error("JSON error: {0}")]
    JsonError(#[from] serde_json::Error),
    #[error("Base64 decode error: {0}")]
    Base64Error(#[from] base64::DecodeError),
}

pub struct Resource {
    resources: HashMap<String, Value>,
}

impl Resource {
    pub fn init() -> Result<Self, ResourceError> {
        let mut resources: HashMap<String, Value> = HashMap::new();

        if let (Ok(sst_key), Ok(sst_key_file)) = (env::var("SST_KEY"), env::var("SST_KEY_FILE")) {
            let key = BASE64_STANDARD.decode(sst_key)?;
            let encrypted_data = std::fs::read(sst_key_file)?;

            let nonce = GenericArray::from_slice(&[0u8; 12]);
            let cipher = Aes256Gcm::new(GenericArray::from_slice(&key));

            let auth_tag_start = encrypted_data.len() - 16;
            let actual_ciphertext = &encrypted_data[..auth_tag_start];
            let auth_tag = &encrypted_data[auth_tag_start..];

            let mut ciphertext_with_tag = Vec::with_capacity(encrypted_data.len());
            ciphertext_with_tag.extend_from_slice(actual_ciphertext);
            ciphertext_with_tag.extend_from_slice(auth_tag);

            let decrypted = cipher
                .decrypt(nonce, ciphertext_with_tag.as_ref())
                .map_err(|e| ResourceError::DecryptionError(e.to_string()))?;

            resources = serde_json::from_slice(&decrypted)?;
        }

        for (key, value) in env::vars() {
            if key.starts_with("SST_RESOURCE_") {
                let result: Value = serde_json::from_str(&value)?;
                resources.insert(key.trim_start_matches("SST_RESOURCE_").to_string(), result);
            }
        }

        Ok(Self { resources })
    }

    pub fn get<D: DeserializeOwned>(&self, name: &str) -> Result<D, ResourceError> {
        let value = self.resources.get(name).ok_or(ResourceError::NotFound)?;

        Ok(serde_json::from_value(value.clone())?)
    }

    pub fn into_inner(self) -> HashMap<String, Value> {
        self.resources
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use serial_test::serial;
    use std::env;
    use std::fs::File;
    use std::io::Write;
    use tempfile::TempDir;

    // Helper to clear all SST-related environment variables
    fn clear_env_vars() {
        env::remove_var("SST_KEY");
        env::remove_var("SST_KEY_FILE");
        
        let sst_resource_vars: Vec<String> = env::vars()
            .filter(|(key, _)| key.starts_with("SST_RESOURCE_"))
            .map(|(key, _)| key)
            .collect();
        
        for var in sst_resource_vars {
            env::remove_var(&var);
        }
    }

    // Helper to create an encrypted file for testing
    fn create_encrypted_file(data: &HashMap<String, Value>) -> (TempDir, String, String) {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("encrypted.bin");

        let key = [0u8; 32];
        let key_base64 = BASE64_STANDARD.encode(&key);

        let json_data = serde_json::to_vec(data).unwrap();
        let nonce = GenericArray::from_slice(&[0u8; 12]);
        let cipher = Aes256Gcm::new(GenericArray::from_slice(&key));

        let ciphertext = cipher.encrypt(nonce, json_data.as_ref()).unwrap();

        let mut file = File::create(&file_path).unwrap();
        file.write_all(&ciphertext).unwrap();

        (temp_dir, file_path.to_str().unwrap().to_string(), key_base64)
    }

    #[test]
    #[serial]
    fn test_init_without_sst_key_vars() {
        clear_env_vars();

        let resource = Resource::init();
        assert!(resource.is_ok());
        let resource = resource.unwrap();
        assert_eq!(resource.resources.len(), 0);
    }

    #[test]
    #[serial]
    fn test_init_with_sst_resource_env_vars() {
        clear_env_vars();

        env::set_var("SST_RESOURCE_MyBucket", r#"{"name":"my-bucket","type":"aws.s3.Bucket"}"#);
        env::set_var("SST_RESOURCE_MyTable", r#"{"name":"my-table","type":"aws.dynamodb.Table"}"#);

        let resource = Resource::init().unwrap();

        assert_eq!(resource.resources.len(), 2);
        assert!(resource.resources.contains_key("MyBucket"));
        assert!(resource.resources.contains_key("MyTable"));
    }

    #[test]
    #[serial]
    fn test_init_with_encrypted_file() {
        clear_env_vars();

        let mut data = HashMap::new();
        data.insert("EncryptedBucket".to_string(), json!({"name": "encrypted-bucket", "type": "aws.s3.Bucket"}));
        data.insert("EncryptedQueue".to_string(), json!({"name": "encrypted-queue", "type": "aws.sqs.Queue"}));

        let (_temp_dir, file_path, key_base64) = create_encrypted_file(&data);

        env::set_var("SST_KEY", &key_base64);
        env::set_var("SST_KEY_FILE", &file_path);

        let resource = Resource::init().unwrap();

        assert_eq!(resource.resources.len(), 2);
        assert!(resource.resources.contains_key("EncryptedBucket"));
        assert!(resource.resources.contains_key("EncryptedQueue"));
    }

    #[test]
    #[serial]
    fn test_init_with_both_encrypted_and_env_vars() {
        clear_env_vars();

        let mut encrypted_data = HashMap::new();
        encrypted_data.insert("EncryptedResource".to_string(), json!({"name": "encrypted", "type": "aws.s3.Bucket"}));

        let (_temp_dir, file_path, key_base64) = create_encrypted_file(&encrypted_data);

        env::set_var("SST_KEY", &key_base64);
        env::set_var("SST_KEY_FILE", &file_path);
        env::set_var("SST_RESOURCE_EnvResource", r#"{"name":"env-resource","type":"aws.dynamodb.Table"}"#);

        let resource = Resource::init().unwrap();

        assert_eq!(resource.resources.len(), 2);
        assert!(resource.resources.contains_key("EncryptedResource"));
        assert!(resource.resources.contains_key("EnvResource"));
    }

    #[test]
    #[serial]
    fn test_get_existing_resource() {
        clear_env_vars();

        env::set_var("SST_RESOURCE_TestBucket", r#"{"name":"test-bucket","arn":"arn:aws:s3:::test-bucket"}"#);

        let resource = Resource::init().unwrap();

        #[derive(serde::Deserialize, Debug, PartialEq)]
        struct BucketInfo {
            name: String,
            arn: String,
        }

        let bucket: BucketInfo = resource.get("TestBucket").unwrap();
        assert_eq!(bucket.name, "test-bucket");
        assert_eq!(bucket.arn, "arn:aws:s3:::test-bucket");
    }

    #[test]
    #[serial]
    fn test_get_nonexistent_resource() {
        clear_env_vars();

        let resource = Resource::init().unwrap();

        let result: Result<Value, _> = resource.get("NonExistent");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ResourceError::NotFound));
    }

    #[test]
    #[serial]
    fn test_init_with_only_sst_key() {
        clear_env_vars();

        env::set_var("SST_KEY", "dGVzdA=="); // base64 "test"

        let resource = Resource::init();
        assert!(resource.is_ok());
    }

    #[test]
    #[serial]
    fn test_into_inner() {
        clear_env_vars();

        env::set_var("SST_RESOURCE_Resource1", r#"{"value":"test1"}"#);
        env::set_var("SST_RESOURCE_Resource2", r#"{"value":"test2"}"#);

        let resource = Resource::init().unwrap();
        let inner = resource.into_inner();

        assert_eq!(inner.len(), 2);
        assert!(inner.contains_key("Resource1"));
        assert!(inner.contains_key("Resource2"));
    }
}