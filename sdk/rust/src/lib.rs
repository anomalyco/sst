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
        let key = BASE64_STANDARD.decode(env::var("SST_KEY")?)?;
        let encrypted_data = std::fs::read(env::var("SST_KEY_FILE")?)?;

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

        let mut resources: HashMap<String, Value> = serde_json::from_slice(&decrypted)?;

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
