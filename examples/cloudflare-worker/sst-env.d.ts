/* tslint:disable */
/* eslint-disable */
import "sst"
declare module "sst" {
  export interface Resource {
    MyWorker: {
      type: "sst.cloudflare.Worker"
      url: string
    }
  }
}
// cloudflare 
declare module "sst" {
  export interface Resource {
    MyBucket: import("@cloudflare/workers-types").R2Bucket
  }
}
