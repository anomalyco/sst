import { ping } from "@repo/lib/ping";
import type { Handler } from "aws-lambda";
import { Resource } from "sst";

export const handler: Handler = async (event) => {
    // Call the ping function to get response code
    const responseCode = await ping();
    console.log(`Response code: ${responseCode}`);

    return {
        statusCode: 200,
        body: `Hello from Node.js! - Linkable value: ${Resource.MyLinkableValue.foo}`,
    };
};
