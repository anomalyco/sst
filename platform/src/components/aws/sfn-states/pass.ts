import { Input } from '@pulumi/pulumi';

import { StateBase, StateBaseParams } from './state';

export interface PassParams extends StateBaseParams {
  Result?: Input<Record<string, unknown>>;
  Parameters?: Input<Record<string, unknown>>;
}
export class Pass extends StateBase {
  constructor(
    public name: string,
    protected params: PassParams = {}
  ) {
    super(name, params);
  }
  toJSON() {
    return {
      ...super.toJSON(),
      Type: 'Pass',
      ...this.params,
    };
  }
}
