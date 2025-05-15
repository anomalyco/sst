export const MyResource = sst.resource({
  type: "MyResource",
  async create(inputs: { butt: number }) {
    return {
      id: "123",
      outputs: {
        hello: "world",
        updated: Date.now(),
      },
    };
  },
  async delete(id, inputs) {
    console.log("remove");
  },
  async update(id, state, news) {
    return {
      ...state.outputs,
      updated: Date.now(),
    };
  },
});
