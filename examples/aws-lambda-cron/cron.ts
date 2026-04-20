export async function handler(event: any) {
  console.log("Cron triggered", JSON.stringify(event));
  return { statusCode: 200 };
}
