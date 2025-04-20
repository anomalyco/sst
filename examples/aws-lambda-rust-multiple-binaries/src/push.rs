use std::time::Duration;

use aws_sdk_s3::presigning::PresigningConfig;
use lambda_runtime::{service_fn, Error, LambdaEvent};
use serde::Deserialize;
use serde_json::Value;
use sst_sdk::Resource;

#[derive(Deserialize, Debug)]
struct Bucket {
    name: String,
}

async fn presigned_url(_event: LambdaEvent<Value>) -> Result<String, Error> {
    let config = aws_config::load_from_env().await;
    let client = aws_sdk_s3::Client::new(&config);
    let resource = Resource::init().unwrap();
    let Bucket { name } = resource.get("Bucket").unwrap();

    let url = client
        .put_object()
        .bucket(name)
        .key(uuid::Uuid::new_v4())
        .presigned(
            PresigningConfig::builder()
                .expires_in(Duration::from_secs(60 * 10))
                .build()
                .unwrap(),
        )
        .await
        .unwrap();

    Ok(url.uri().to_string())
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    let func = service_fn(presigned_url);
    lambda_runtime::run(func).await?;
    Ok(())
}
