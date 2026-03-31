import { Resource } from "sst";

export const handler = async (event) => {
  console.log("event", event);
  console.log("Resource.MyBucket", Resource.MyBucket);
};
