export const create = async (event) => {
  console.log("create", event);
  return "ok";
};

export const update = async (event) => {
  console.log("update", event);
  return "ok";
};

export const remove = async (event) => {
  console.log("delete", event);
  return "ok";
};
