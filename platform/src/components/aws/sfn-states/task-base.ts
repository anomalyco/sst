import { Input } from "@pulumi/pulumi";

import { Retriable, StateBase, StateBaseParams } from "./state";
import type aws from "@pulumi/aws";

export interface TaskStateBaseParams<TP> extends StateBaseParams {
  Resource: Input<string>;
  Parameters: TP;
}

export class TaskStateBase<TP> extends StateBase implements Retriable {
  constructor(
    public name: string,
    protected params: TaskStateBaseParams<TP>,
  ) {
    super(name, params);
  }

  toJSON() {
    return {
      ...super.toJSON(),
      Type: "Task",
      ...this.params,
    };
  }

  createPermissions(role: aws.iam.Role, prefix: string) {
    super.createPermissions(role, prefix);
  }
}
