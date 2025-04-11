import * as aws from '@pulumi/aws';
import { Input } from '@pulumi/pulumi';

import { TaskStateBase, TaskStateBaseParams } from './task-base';

type LambdaInvokeTaskParameters = {
  FunctionName: Input<string>;
  Payload?: Record<string, Input<unknown>>;
};

type Parameters = Omit<
  TaskStateBaseParams<LambdaInvokeTaskParameters>,
  'Resource'
> &
  Partial<Pick<TaskStateBaseParams<LambdaInvokeTaskParameters>, 'Resource'>>;

export class LambdaInvoke extends TaskStateBase<LambdaInvokeTaskParameters> {
  constructor(
    public name: string,
    params: Parameters
  ) {
    super(name, {
      Resource: `arn:aws:states:::lambda:invoke`,
      ...params,
    });
  }

  private createRolePolicy(role: aws.iam.Role, prefix: string) {
    new aws.iam.RolePolicy(`${prefix}${this.name}SfnRolePolicy`, {
      role: role.id,
      policy: {
        Version: '2012-10-17',
        Statement: [
          {
            Effect: 'Allow',
            Action: ['lambda:InvokeFunction'],
            Resource: $output(this.params).apply(
              (p) => p.Parameters.FunctionName
            ),
          },
        ],
      },
    });
  }

  createPermissions(role: aws.iam.Role, prefix: string) {
    super.createPermissions(role, prefix);
    this.createRolePolicy(role, prefix);
  }

  toJSON() {
    return {
      ...super.toJSON(),
    };
  }
}
