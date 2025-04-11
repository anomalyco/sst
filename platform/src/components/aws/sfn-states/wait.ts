import { Input } from '@pulumi/pulumi';

import { StateBase, StateBaseParams } from './state';

export interface WaitParams extends StateBaseParams {
  /**
   * The number of seconds to wait.
   * @example 10
   */
  Seconds?: Input<number>;
  /**
   * The timestamp to wait for.
   * @example '2024-03-14T01:59:00Z'
   */
  Timestamp?: Input<string>;
  /**
   * The path to the timestamp.
   * @example $.timestamp
   */
  TimestampPath?: Input<string>;
}
export class Wait extends StateBase {
  constructor(
    public name: string,
    protected params: WaitParams = {}
  ) {
    super(name, params);
  }
  toJSON() {
    return {
      ...super.toJSON(),
      Type: 'Wait',
      ...this.params,
    };
  }
}
