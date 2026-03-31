import { Resource } from "sst";
import { Example } from "@aws-auth-react/core/example";

console.log(`${Example.hello()} Linked to ${Resource.MyBucket.name}.`);
