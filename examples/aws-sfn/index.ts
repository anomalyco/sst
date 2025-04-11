import { Handler } from "aws-lambda";

type Event = {
  name: string;
};

type Result = {
  status: "COMPLETED" | "FAILED";
};

export const handler: Handler<Event, Result> = async (event) => {
  console.log(event);

  return {
    status: Math.random() < 0.5 ? "COMPLETED" : "FAILED",
  };
};
