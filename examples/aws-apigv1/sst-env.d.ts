/* tslint:disable */
/* eslint-disable */
import "sst"
declare module "sst" {
  export interface Resource {
    MyApi: {
      type: "sst.aws.ApiGatewayV1"
      url: string
    }
    MyApiAuthorizerMyAuthorizerFunction: {
      name: string
      type: "sst.aws.Function"
    }
  }
}
export {}