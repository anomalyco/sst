import { Resource } from "../resource.js";
import { client } from "../aws/client.js";

export interface PutEvent {
  /**
   * The vector to store in the database.
   * @example
   * ```js
   * {
   *   vector: [32.4, 6.55, 11.2, 10.3, 87.9]
   * }
   * ```
   */
  vector: number[];
  /**
   * Metadata for the event as JSON.
   * This will be used to filter when querying and removing vectors.
   * @example
   * ```js
   * {
   *   metadata: {
   *     type: "movie",
   *     id: "movie-123",
   *     name: "Spiderman"
   *   }
   * }
   * ```
   */
  metadata: Record<string, any>;
}

export interface QueryEvent {
  /**
   * The vector used to query the database.
   * @example
   * ```js
   * {
   *   vector: [32.4, 6.55, 11.2, 10.3, 87.9]
   * }
   * ```
   */
  vector: number[];
  /**
   * The metadata used to filter the vectors.
   * Only vectors that match the provided fields will be returned.
   * @example
   * Given this filter.
   * ```js
   * {
   *   include: {
   *     type: "movie",
   *     release: "2001"
   *   }
   * }
   * ```
   * It will match a vector with the metadata:
   * ```js
   * {
   *   type: "movie",
   *   name: "Spiderman",
   *   release: "2001"
   * }
   * ```
   *
   * But not a vector with this metadata:
   * ```js
   *  {
   *    type: "book",
   *    name: "Spiderman",
   *    release: "2001"
   *  }
   * ```
   */
  include: Record<string, any>;
  /**
   * Exclude vectors with metadata that match the provided fields.
   * @example
   * Given this filter.
   * ```js
   * {
   *   include: {
   *     type: "movie",
   *     release: "2001"
   *   },
   *   exclude: {
   *     name: "Spiderman"
   *   }
   * }
   * ```
   * This will match a vector with metadata:
   * ```js
   *  {
   *    type: "movie",
   *    name: "A Beautiful Mind",
   *    release: "2001"
   *  }
   * ```
   *
   * But not a vector with the metadata:
   * ```js
   *  {
   *    type: "book",
   *    name: "Spiderman",
   *    release: "2001"
   *  }
   * ```
   */
  exclude?: Record<string, any>;
  /**
   * The threshold of similarity between the prompt and the queried vectors.
   * Only vectors with a similarity score higher than the threshold will be returned.
   *
   * This will return values is between 0 and 1.
   * - `0` means the prompt and the queried vectors are completely different.
   * - `1` means the prompt and the queried vectors are identical.
   *
   * @default `0`
   * @example
   * ```js
   * {
   *   threshold: 0.5
   * }
   * ```
   */
  threshold?: number;
  /**
   * The number of results to return.
   * @default `10`
   * @example
   * ```js
   * {
   *   count: 10
   * }
   * ```
   */
  count?: number;
}

export interface RemoveEvent {
  /**
   * The metadata used to filter the removal of vectors.
   * Only vectors with metadata that match the provided fields will be removed.
   * @example
   * To remove vectors for movie with id `movie-123`:
   * ```js
   * {
   *   include: {
   *     id: "movie-123",
   *   }
   * }
   * ```
   * To remove vectors for all _movies_:
   * ```js
   *  {
   *   include: {
   *    type: "movie",
   *   }
   *  }
   * ```
   */
  include: Record<string, any>;
}

export interface QueryResponse {
  /**
   * List of results matching the query.
   */
  results: {
    /**
     * Metadata for the event that was provided when storing the vector.
     */
    metadata: Record<string, any>;
    /**
     * The similarity score between the prompt and the queried vector.
     */
    score: number;
  }[];
}

export interface VectorClientResponse {
  /**
   * Store a vector into the database.
   * @example
   * ```ts title="src/lambda.ts"
   * await client.put({
   *   vector: [32.4, 6.55, 11.2, 10.3, 87.9],
   *   metadata: { type: "movie", genre: "comedy" },
   * });
   * ```
   */
  put: (event: PutEvent) => Promise<void>;
  /**
   * Query vectors that are similar to the given vector
   * @example
   * ```ts title="src/lambda.ts"
   * const result = await client.query({
   *   vector: [32.4, 6.55, 11.2, 10.3, 87.9],
   *   include: { type: "movie" },
   *   exclude: { genre: "thriller" },
   * });
   * ```
   */
  query: (event: QueryEvent) => Promise<QueryResponse>;
  /**
   * Remove vectors from the database.
   * @example
   * ```ts title="src/lambda.ts"
   * await client.remove({
   *   include: { type: "movie" },
   * });
   * ```
   */
  remove: (event: RemoveEvent) => Promise<void>;
}

/**
 * Create a client to interact with the Vector database.
 * @example
 * ```ts title="src/lambda.ts"
 * import { VectorClient } from "sst";
 * const client = VectorClient("MyVectorDB");
 * ```
 *
 * Store a vector into the db
 *
 * ```ts title="src/lambda.ts"
 * await client.put({
 *   vector: [32.4, 6.55, 11.2, 10.3, 87.9],
 *   metadata: { type: "movie", genre: "comedy" },
 * });
 * ```
 *
 * Query vectors that are similar to the given vector
 *
 * ```ts title="src/lambda.ts"
 * const result = await client.query({
 *   vector: [32.4, 6.55, 11.2, 10.3, 87.9],
 *   include: { type: "movie" },
 *   exclude: { genre: "thriller" },
 * });
 * ```
 */
export function VectorClient<
  T extends keyof {
    // @ts-expect-error
    [key in keyof Resource as "sst.aws.Vector" extends Resource[key]["type"]
      ? string extends key
        ? never
        : key
      : never]: Resource[key];
  }
>(name: T): VectorClientResponse {
  return {
    put: async (event: PutEvent) => {
      await invokeFunction(
        // @ts-expect-error
        Resource[name].putFunction,
        JSON.stringify(event),
        "Failed to store into the vector db"
      );
    },

    query: async (event: QueryEvent) => {
      return await invokeFunction<QueryResponse>(
        // @ts-expect-error
        Resource[name].queryFunction,
        JSON.stringify(event),
        "Failed to query the vector db"
      );
    },

    remove: async (event: RemoveEvent) => {
      await invokeFunction(
        // @ts-expect-error
        Resource[name].removeFunction,
        JSON.stringify(event),
        "Failed to remove from the vector db"
      );
    },
  };
}

async function invokeFunction<T>(
  functionName: string,
  body: string,
  errorMessage: string,
  attempts = 0
): Promise<T> {
  try {
    const c = await client();
    const endpoint = `https://lambda.${process.env.AWS_REGION}.amazonaws.com/2015-03-31`;
    const response = await c.fetch(
      `${endpoint}/functions/${functionName}/invocations`,
      {
        method: "POST",
        headers: { Accept: "application/json" },
        body,
      }
    );

    // success
    if (response.status === 200 || response.status === 201) {
      if (response.headers.get("content-length") === "0") return undefined as T;
      const text = await response.text();
      try {
        return JSON.parse(text);
      } catch (e) {
        throw new Error(`Failed to parse JSON response: ${text}`);
      }
    }

    // error
    const error = new Error();
    const text = await response.text();
    try {
      const json = JSON.parse(text);
      error.name = json.Error?.Code;
      error.message = json.Error?.Message ?? json.message ?? text;
    } catch (e) {
      error.message = text;
    }
    error.name = error.name ?? response.headers.get("x-amzn-ErrorType");
    // @ts-expect-error
    error.requestID = response.headers.get("x-amzn-RequestId");
    // @ts-expect-error
    error.statusCode = response.status;
    throw error;
  } catch (e: any) {
    let isRetryable = false;

    // AWS throttling errors => retry
    if (
      [
        "ThrottlingException",
        "Throttling",
        "TooManyRequestsException",
        "OperationAbortedException",
        "TimeoutError",
        "NetworkingError",
      ].includes(e.name)
    ) {
      isRetryable = true;
    }

    if (!isRetryable) throw e;

    // retry
    await new Promise((resolve) =>
      setTimeout(resolve, 1.5 ** attempts * 100 * Math.random())
    );
    return await invokeFunction<T>(
      functionName,
      body,
      errorMessage,
      attempts + 1
    );
  }
}
