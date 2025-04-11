import { Input } from "@pulumi/pulumi";

import { type Chainable, StateBase, StateBaseParams } from "./state";

export interface FailParams extends StateBaseParams {
  Error?: Input<string>;
  Cause?: Input<string>;
  CausePath?: Input<string>;
  ErrorPath?: Input<string>;
}
export class Fail extends StateBase {
  constructor(
    public name: string,
    protected params: FailParams = {},
  ) {
    super(name, params);
  }

  public next(_: Chainable): Chainable {
    throw new Error("Cannot call next on Fail state");
  }

  toJSON() {
    const vals = super.toJSON();
    // @ts-expect-error We should improve the types for the JSON output
    delete vals["End"];
    return {
      ...vals,
      Type: "Fail",
      ...this.params,
    };
  }
}
