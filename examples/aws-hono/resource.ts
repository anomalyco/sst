export const MyResource = sst.resource({
  async create(name, inputs: { foo: string }) {
    const result = await fetch("http://infrastructure.com/create", {
      method: "POST",
      body: JSON.stringify(inputs),
    }).then((res) => res.json());
    return {
      id: result.id,
      outputs: result,
    };
  },
  async delete(name, state) {
    await fetch("http://infrastructure.com/create/" + state.outputs.id, {
      method: "DELETE",
    }).then((res) => res.json());
  },
});

new MyResource("Example", {
  foo: "some-value",
});
