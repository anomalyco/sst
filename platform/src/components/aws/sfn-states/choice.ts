import { Chainable, Retriable, StateBase, StateBaseParams } from "./state";

type ComparisonOperator =
  | "And"
  | "BooleanEquals"
  | "BooleanEqualsPath"
  | "IsBoolean"
  | "IsNull"
  | "IsNumeric"
  | "IsPresent"
  | "IsString"
  | "IsTimestamp"
  | "Not"
  | "NumericEquals"
  | "NumericEqualsPath"
  | "NumericGreaterThan"
  | "NumericGreaterThanPath"
  | "NumericGreaterThanEquals"
  | "NumericGreaterThanEqualsPath"
  | "NumericLessThan"
  | "NumericLessThanPath"
  | "NumericLessThanEquals"
  | "NumericLessThanEqualsPath"
  | "Or"
  | "StringEquals"
  | "StringEqualsPath"
  | "StringGreaterThan"
  | "StringGreaterThanPath"
  | "StringGreaterThanEquals"
  | "StringGreaterThanEqualsPath"
  | "StringLessThan"
  | "StringLessThanPath"
  | "StringLessThanEquals"
  | "StringLessThanEqualsPath"
  | "StringMatches"
  | "TimestampEquals"
  | "TimestampEqualsPath"
  | "TimestampGreaterThan"
  | "TimestampGreaterThanPath"
  | "TimestampGreaterThanEquals"
  | "TimestampGreaterThanEqualsPath"
  | "TimestampLessThan"
  | "TimestampLessThanPath"
  | "TimestampLessThanEquals"
  | "TimestampLessThanEqualsPath";

type ChoiceRule = {
  Variable?: string;
  Condition?: string;
} & Partial<Record<ComparisonOperator, unknown>> & {
    Next: Chainable;
  };

export interface ChoiceParams extends StateBaseParams {
  QueryLanguage?: "JSONata" | "JSONPath";
}

export class Choice extends StateBase implements Retriable {
  private choices: ChoiceRule[] = [];
  private defaultNext?: Chainable;
  private queryLanguage: "JSONata" | "JSONPath" = "JSONata";

  constructor(
    public name: string,
    protected params: ChoiceParams = {},
  ) {
    super(name, params);
    this.queryLanguage = params.QueryLanguage || "JSONata";
  }

  // JSONata methods
  when<T extends `{%${string}%}`>(condition: T, next: Chainable) {
    if (this.queryLanguage !== "JSONata") {
      throw new Error(
        "Cannot use JSONata conditions when QueryLanguage is JSONPath",
      );
    }
    if (!condition.startsWith("{%") || !condition.endsWith("%}")) {
      throw new Error("Condition must start with '{%' and end with '%}'.");
    }
    this.choices.push({ Condition: condition, Next: next });
    return this;
  }

  // JSONPath methods
  whenEquals(variable: string, value: unknown, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Variable: variable, StringEquals: value, Next: next });
    return this;
  }

  whenNumericEquals(variable: string, value: number, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Variable: variable, NumericEquals: value, Next: next });
    return this;
  }

  whenBooleanEquals(variable: string, value: boolean, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Variable: variable, BooleanEquals: value, Next: next });
    return this;
  }

  whenIsNull(variable: string, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Variable: variable, IsNull: true, Next: next });
    return this;
  }

  whenIsPresent(variable: string, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Variable: variable, IsPresent: true, Next: next });
    return this;
  }

  whenStringMatches(variable: string, pattern: string, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({
      Variable: variable,
      StringMatches: pattern,
      Next: next,
    });
    return this;
  }

  whenTimestampEquals(variable: string, timestamp: string, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({
      Variable: variable,
      TimestampEquals: timestamp,
      Next: next,
    });
    return this;
  }

  whenAnd(rules: ChoiceRule[], next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ And: rules, Next: next });
    return this;
  }

  whenOr(rules: ChoiceRule[], next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Or: rules, Next: next });
    return this;
  }

  whenNot(rule: ChoiceRule, next: Chainable) {
    if (this.queryLanguage !== "JSONPath") {
      throw new Error(
        "Cannot use JSONPath conditions when QueryLanguage is JSONata",
      );
    }
    this.choices.push({ Not: rule, Next: next });
    return this;
  }

  otherwise(next: Chainable) {
    this.defaultNext = next;
    return this;
  }

  public next(_: Chainable): Chainable {
    throw new Error("Cannot call next on Choice state");
  }

  serialize() {
    let obj = super.serialize();
    for (const c of this.choices) {
      obj = { ...obj, ...c.Next.serialize() };
    }
    if (this.defaultNext) {
      obj = { ...obj, ...this.defaultNext.serialize() };
    }
    return obj;
  }

  toJSON() {
    const vals = super.toJSON();
    // @ts-expect-error We should improve the types for the JSON output
    delete vals["End"];
    
    return {
      ...vals,
      Choices: this.choices.map((c) => ({
        ...c,
        Next: c.Next.name,
      })),
      Type: "Choice",
      QueryLanguage: this.queryLanguage,
      Default: this.defaultNext?.name,
    };
  }
}
