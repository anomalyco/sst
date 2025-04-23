use std::{env::set_var, time::Duration};

use aws_sdk_s3::presigning::PresigningConfig;
use axum::{routing::get, Router};
use lambda_http::{run, tracing, Error};
use serde::Deserialize;
use sst_sdk::Resource;

#[derive(Deserialize, Debug)]
struct Bucket {
    name: String,
}

async fn presigned_url() -> String {
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

    url.uri().to_string()
}

async fn latest() -> String {
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

    url.uri().to_string()
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    // If you use API Gateway stages, the Rust Runtime will include the stage name
    // as part of the path that your application receives.
    // Setting the following environment variable, you can remove the stage from the path.
    // This variable only applies to API Gateway stages,
    // you can remove it if you don't use them.
    // i.e with: `GET /test-stage/todo/id/123` without: `GET /todo/id/123`
    set_var("AWS_LAMBDA_HTTP_IGNORE_STAGE_IN_PATH", "true");

    tracing::init_default_subscriber();

    let app = Router::new()
        .route("/", get(presigned_url))
        .route("/latest", get(latest));

    run(app).await
}
