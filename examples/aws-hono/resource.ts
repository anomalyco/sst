export const MyResource = sst.resource({
  async create(name, inputs: { butt: number }) {
    return {
      id: "123",
      outputs: {
        hello: "world",
        updated: Date.now(),
      },
    };
  },
  async update(name, olds, news) {
    console.log(name, olds, news);
    return {
      ...olds.outputs,
      updated: Date.now(),
    };
  },
});
