import { ZodSchema, z } from "zod";

export function ZodValidator<Schema extends ZodSchema>(
  schema: Schema,
): (input: z.input<Schema>) => z.output<Schema> {
  return (input) => {
    return schema.parse(input);
  };
}

import { parse, BaseSchema, InferInput } from "valibot";

export function ValibotValidator<T extends BaseSchema<any, any, any>>(
  schema: T,
) {
  return (value: InferInput<T>) => {
    return parse(schema, value);
  };
}
