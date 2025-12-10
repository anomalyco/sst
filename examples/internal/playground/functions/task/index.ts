import { Resource } from "/Users/frank/Sites/sst/sdk/js/src/resource";
import { task } from "/Users/frank/Sites/sst/sdk/js/src/aws/task";

export const handler = async () => {
  const ret = await task.run(Resource.MyTask);

  if (ret.response.tasks?.length) {
    return {
      taskArn: ret.response.tasks[0].taskArn,
    };
  }

  return JSON.stringify(
    {
      ARN: ret.response.tasks[0]?.taskArn,
      response: ret.response,
    },
    null,
    2
  );

  //const ret = await task.describe(Resource.MyTask, t);

  //const ret = await task.stop(Resource.MyTask, t);
  //console.log(ret.task?.lastStatus);
};
