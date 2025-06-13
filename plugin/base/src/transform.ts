import { runtime } from "@pulumi/pulumi";
import { ComponentTransforms } from "./component";

export function transform<T, Args, Options>(
  resource: { new (name: string, args: Args, opts?: Options): T },
  cb: (args: Args, opts: Options, name: string) => void,
) {
  // @ts-expect-error
  const type = resource.__pulumiType;
  if (type.startsWith("sst:")) {
    let transforms = ComponentTransforms.get(type);
    if (!transforms) {
      transforms = [];
      ComponentTransforms.set(type, transforms);
    }
    transforms.push((input: any) => {
      cb(input.props, input.opts, input.name);
      return input;
    });
    return;
  }
  runtime.registerStackTransformation((input) => {
    if (input.type !== type) return;
    cb(input.props as any, input.opts as any, input.name);
    return input;
  });
}
