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

async fn latest(_event: LambdaEvent<Value>) -> Result<String, Error> {
    let config = aws_config::load_from_env().await;
    let client = aws_sdk_s3::Client::new(&config);
    let resource = Resource::init().unwrap();
    let Bucket { name } = resource.get("Bucket").unwrap();

    let objects = client.list_objects().bucket(&name).send().await.unwrap();
    let latest = objects
        .contents()
        .into_iter()
        .min_by_key(|o| o.last_modified().unwrap())
        .unwrap();

    let url = client
        .get_object()
        .bucket(name)
        .key(latest.key().unwrap())
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
    let func = service_fn(latest);
    lambda_runtime::run(func).await?;
    Ok(())
}
