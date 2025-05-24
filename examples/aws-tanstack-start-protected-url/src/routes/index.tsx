import { createFileRoute } from "@tanstack/react-router";
import { createServerFn } from "@tanstack/react-start";

const greetUser = createServerFn({ method: "POST" })
  .validator((data: any) => {
    if (!data.name || !data.age) {
      throw new Error("Name and age are required");
    }

    return {
      name: String(data.name),
      age: parseInt(data.age, 10),
    };
  })
  .handler(async ({ data: { name, age } }) => {
    return `Hello, ${name}! You are ${age} years old.`;
  });

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  return (
    <div className="p-2">
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          const formData = new FormData(event.currentTarget);
          const name = formData.get("name");
          const age = formData.get("age");
          const response = await greetUser({ data: { name, age } });
          console.log(response);
        }}
      >
        <input name="name" placeholder="Name" />
        <input name="age" placeholder="Age" />
        <button type="submit">Submit</button>
      </form>
    </div>
  );
}
