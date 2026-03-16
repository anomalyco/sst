/* tslint:disable */
/* eslint-disable */
import "sst"
declare module "sst" {
  export interface Resource {
    Database: {
      host: string
      username: string
      password: string
      database: string
      port: number
      type: "sst.sst.Linkable"
    }
    Api: {
      name: string
      type: "sst.aws.Function"
      url: string
    }
  }
}
export {}
