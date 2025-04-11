import { StateBase, StateBaseParams } from './state';

export interface SucceedParams extends StateBaseParams {}
export class Succeed extends StateBase {
  constructor(
    public name: string,
    protected params: SucceedParams = {}
  ) {
    super(name, params);
  }

  toJSON() {
    const vals = super.toJSON();
    // @ts-expect-error We should improve the types for the JSON output
    delete vals['End'];
    return {
      ...vals,
      Type: 'Succeed',
    };
  }
}
