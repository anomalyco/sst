using SST;

var builder = WebApplication.CreateBuilder(args);

// On Lambda this replaces Kestrel with the Lambda runtime and translates API
// Gateway V2 events into HTTP requests. Locally, `dotnet run` serves on Kestrel.
builder.Services.AddAWSLambdaHosting(LambdaEventSource.HttpApi);

var app = builder.Build();

app.MapGet("/", () => Results.Json(new { message = "home page", app = Resource.App.Name, stage = Resource.App.Stage }));
app.MapGet("/hello", () => Results.Json(new { message = "hello world" }));
app.MapGet("/my-ping", () => Results.Json(new { message = "pong" }));
app.MapFallback(() => Results.Json(new { message = "not found" }, statusCode: StatusCodes.Status404NotFound));

app.Run();
