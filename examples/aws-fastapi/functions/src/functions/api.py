from fastapi import FastAPI
from mangum import Mangum
from sst import Resource
from functions.shared import foo
from core.db import get_user
from core.ping import ping

# Create FastAPI app with JSON configuration
app = FastAPI(
    title="SST FastAPI App",
    json_encoder=None  # Use FastAPI's default JSON encoder
)

# Create a route
@app.get("/")
async def root():
    print("Function invoked from Python")

    # Share code within the same workspace package
    result = foo()
    print(result)

    # Share code between workspace packages
    res = ping()
    user = get_user("1234")
    print(user)
    print("Ping result:", res)

    # Use the SST SDK to access resources
    resource_value = Resource.MyLinkableValue.foo
    print(f"Resource.MyLinkableValue.foo: {resource_value}")

    return {
        "message": f"{resource_value} from Python!",
        
    }

# Create handler for AWS Lambda with specific configuration
handler = Mangum(app, api_gateway_base_path=None, lifespan="off")
