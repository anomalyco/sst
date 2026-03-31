import { Resource } from "sst";

export const authorizer = async () => {
  return {
    isAuthorized: true,
    context: {
      userId: "123",
    },
  };
};

export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ event, resources: Resource.MyBucket }, null, 2),
  };
};
