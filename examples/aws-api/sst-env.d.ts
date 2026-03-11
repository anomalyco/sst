/* tslint:disable */
/* eslint-disable */
import "sst"
declare module "sst" {
  export interface Resource {
    MyApi: {
      type: "sst.aws.ApiGatewayV2"
      url: string
    }
    MyBucket: {
      name: string
      type: "sst.aws.Bucket"
    }
  }
}
export {}
