#:property TargetFramework=net10.0
#:property PublishAot=false
#:package Amazon.Lambda.Core@3.3.0
#:package Amazon.Lambda.RuntimeSupport@2.2.0
#:package Amazon.Lambda.APIGatewayEvents@3.0.1
#:package Amazon.Lambda.Serialization.SystemTextJson@3.0.1
#:package AWSSDK.DynamoDBv2@4.0.103.7
// this will break when not in this repo.
#:project ../../../sdk/csharp/src/SST.Sdk.csproj

using System.Net;
using System.Text;
using System.Text.Json;
using Amazon.DynamoDBv2;
using Amazon.DynamoDBv2.Model;
using Amazon.Lambda.APIGatewayEvents;
using Amazon.Lambda.Core;
using Amazon.Lambda.RuntimeSupport;
using Amazon.Lambda.Serialization.SystemTextJson;
using SST;

// The table name comes from `link: [table]` in sst.config.ts. Reading it at startup
// means a broken link fails the cold start instead of the first request.
var tableName = Resource.Get<string>("DotnetTable", "name");
var dynamo = new Lazy<IAmazonDynamoDB>(() => new AmazonDynamoDBClient());
var json = new JsonSerializerOptions(JsonSerializerDefaults.Web);

// Function URLs use the same payload as API Gateway V2, so its request type fits.
Func<APIGatewayHttpApiV2ProxyRequest, ILambdaContext, Task<APIGatewayHttpApiV2ProxyResponse>> handler =
    async (request, context) =>
    {
        var method = request.RequestContext?.Http?.Method ?? "GET";
        var query = request.QueryStringParameters ?? new Dictionary<string, string>();

        try
        {
            return (method, request.RawPath) switch
            {
                ("GET", "/") => Json(HttpStatusCode.OK, new
                {
                    message = "notes api",
                    app = Resource.App.Name,
                    stage = Resource.App.Stage,
                    table = tableName,
                }),
                ("POST", "/notes") => await CreateNote(Body(request)),
                ("GET", "/notes") when query.ContainsKey("noteId") => await GetNote(query),
                ("GET", "/notes") => await ListNotes(query),
                ("DELETE", "/notes") => await DeleteNote(query),
                _ => Json(HttpStatusCode.NotFound, new { message = "not found" }),
            };
        }
        catch (ArgumentException e)
        {
            return Json(HttpStatusCode.BadRequest, new { message = e.Message });
        }
    };

// Executable assemblies talk to the Lambda Runtime API themselves. That is what
// lets `sst dev` run the same file locally.
await LambdaBootstrapBuilder.Create(handler, new DefaultLambdaJsonSerializer())
    .Build()
    .RunAsync();

async Task<APIGatewayHttpApiV2ProxyResponse> CreateNote(string body)
{
    var input = JsonSerializer.Deserialize<NoteInput>(body, json);
    if (string.IsNullOrWhiteSpace(input?.UserId) || string.IsNullOrWhiteSpace(input.Content))
    {
        throw new ArgumentException("userId and content are required");
    }

    var note = new Note(input.UserId, Guid.NewGuid().ToString("N"), input.Content, DateTimeOffset.UtcNow.ToString("O"));
    await dynamo.Value.PutItemAsync(new PutItemRequest
    {
        TableName = tableName,
        Item = new Dictionary<string, AttributeValue>
        {
            ["userId"] = new(note.UserId),
            ["noteId"] = new(note.NoteId),
            ["content"] = new(note.Content),
            ["createdAt"] = new(note.CreatedAt),
        },
    });
    return Json(HttpStatusCode.Created, note);
}

async Task<APIGatewayHttpApiV2ProxyResponse> ListNotes(IDictionary<string, string> query)
{
    var userId = Require(query, "userId");
    var response = await dynamo.Value.QueryAsync(new QueryRequest
    {
        TableName = tableName,
        KeyConditionExpression = "userId = :userId",
        ExpressionAttributeValues = new Dictionary<string, AttributeValue> { [":userId"] = new(userId) },
    });
    var notes = (response.Items ?? []).Select(ToNote).ToList();
    return Json(HttpStatusCode.OK, new { userId, count = notes.Count, notes });
}

async Task<APIGatewayHttpApiV2ProxyResponse> GetNote(IDictionary<string, string> query)
{
    var response = await dynamo.Value.GetItemAsync(new GetItemRequest
    {
        TableName = tableName,
        Key = Key(query),
    });
    return response.Item is { Count: > 0 } item
        ? Json(HttpStatusCode.OK, ToNote(item))
        : Json(HttpStatusCode.NotFound, new { message = "note not found" });
}

async Task<APIGatewayHttpApiV2ProxyResponse> DeleteNote(IDictionary<string, string> query)
{
    await dynamo.Value.DeleteItemAsync(new DeleteItemRequest
    {
        TableName = tableName,
        Key = Key(query),
    });
    return Json(HttpStatusCode.OK, new { message = "note deleted" });
}

Dictionary<string, AttributeValue> Key(IDictionary<string, string> query) => new()
{
    ["userId"] = new(Require(query, "userId")),
    ["noteId"] = new(Require(query, "noteId")),
};

static string Require(IDictionary<string, string> query, string name) =>
    query.TryGetValue(name, out var value) && !string.IsNullOrWhiteSpace(value)
        ? value
        : throw new ArgumentException($"{name} query parameter is required");

static string Body(APIGatewayHttpApiV2ProxyRequest request) =>
    request.Body is null ? ""
    : request.IsBase64Encoded ? Encoding.UTF8.GetString(Convert.FromBase64String(request.Body))
    : request.Body;

static Note ToNote(Dictionary<string, AttributeValue> item) => new(
    item["userId"].S,
    item["noteId"].S,
    item.GetValueOrDefault("content")?.S ?? "",
    item.GetValueOrDefault("createdAt")?.S ?? "");

APIGatewayHttpApiV2ProxyResponse Json(HttpStatusCode status, object body) => new()
{
    StatusCode = (int)status,
    Headers = new Dictionary<string, string> { ["Content-Type"] = "application/json" },
    Body = JsonSerializer.Serialize(body, json),
};

record NoteInput(string? UserId, string? Content);

record Note(string UserId, string NoteId, string Content, string CreatedAt);
