"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.Resource = exports.Linkable = void 0;
exports.env = env;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("./component");
/**
 * The `Linkable` component and the `Linkable.wrap` method lets you link any resources in your
 * app; not just the built-in SST components. It also lets you modify the links SST creates.
 *
 * @example
 *
 * #### Linking any value
 *
 * The `Linkable` component takes a list of properties that you want to link. These can be
 * outputs from other resources or constants.
 *
 * ```ts title="sst.config.ts"
 * new sst.Linkable("MyLinkable", {
 *   properties: { foo: "bar" }
 * });
 * ```
 *
 * You can also use this to combine multiple resources into a single linkable resource. And
 * optionally include permissions or bindings for the linked resource.
 *
 * ```ts title="sst.config.ts"
 * const bucketA = new sst.aws.Bucket("MyBucketA");
 * const bucketB = new sst.aws.Bucket("MyBucketB");
 *
 * const storage = new sst.Linkable("MyStorage", {
 *   properties: {
 *     foo: "bar",
 *     bucketA: bucketA.name,
 *     bucketB: bucketB.name
 *   },
 *   include: [
 *     sst.aws.permission({
 *       actions: ["s3:*"],
 *       resources: [bucketA.arn, bucketB.arn]
 *     })
 *   ]
 * });
 * ```
 *
 * You can now link this resource to your frontend or a function.
 *
 * ```ts title="sst.config.ts" {3}
 * new sst.aws.Function("MyApi", {
 *   handler: "src/lambda.handler",
 *   link: [storage]
 * });
 * ```
 *
 * Then use the [SDK](/docs/reference/sdk/) to access it at runtime.
 *
 * ```js title="src/lambda.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyStorage.bucketA);
 * ```
 *
 * #### Linking any resource
 *
 * You can also wrap any Pulumi Resource class to make it linkable.
 *
 * ```ts title="sst.config.ts"
 * sst.Linkable.wrap(aws.dynamodb.Table, (table) => ({
 *   properties: { tableName: table.name },
 *   include: [
 *     sst.aws.permission({
 *       actions: ["dynamodb:*"],
 *       resources: [table.arn]
 *     })
 *   ]
 * }));
 * ```
 *
 * Now you create an instance of `aws.dynamodb.Table` and link it in your app like any other SST
 * component.
 *
 * ```ts title="sst.config.ts" {7}
 * const table = new aws.dynamodb.Table("MyTable", {
 *   attributes: [{ name: "id", type: "S" }],
 *   hashKey: "id"
 * });
 *
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [table]
 * });
 * ```
 *
 * And use the [SDK](/docs/reference/sdk/) to access it at runtime.
 *
 * ```js title="app/page.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyTable.tableName);
 * ```
 *
 * Your function will also have the permissions defined above.
 *
 * #### Modify built-in links
 *
 * You can also modify how SST creates links. For example, you might want to change the
 * permissions of a linkable resource.
 *
 * ```ts title="sst.config.ts" "sst.aws.Bucket"
 *  sst.Linkable.wrap(sst.aws.Bucket, (bucket) => ({
 *    properties: { name: bucket.name },
 *    include: [
 *      sst.aws.permission({
 *        actions: ["s3:GetObject"],
 *        resources: [bucket.arn]
 *      })
 *    ]
 *  }));
 * ```
 *
 * This overrides the built-in link and lets you create your own.
 */
var Linkable = /** @class */ (function (_super) {
    __extends(Linkable, _super);
    function Linkable(name, definition) {
        var _this = _super.call(this, "sst:sst:Linkable", name, definition, {}) || this;
        _this._name = name;
        _this._definition = definition;
        return _this;
    }
    Object.defineProperty(Linkable.prototype, "name", {
        get: function () {
            return (0, pulumi_1.output)(this._name);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Linkable.prototype, "properties", {
        get: function () {
            return this._definition.properties;
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Linkable.prototype.getSSTLink = function () {
        return this._definition;
    };
    /**
     * Wrap any resource class to make it linkable. Behind the scenes this modifies the
     * prototype of the given class.
     *
     * :::tip
     * Use `Linkable.wrap` to make any resource linkable.
     * :::
     *
     * @param cls The resource class to wrap.
     * @param cb A callback that returns the definition for the linkable resource.
     *
     * @example
     *
     * Here we are wrapping the [`aws.dynamodb.Table`](https://www.pulumi.com/registry/packages/aws/api-docs/dynamodb/table/)
     * class to make it linkable.
     *
     * ```ts title="sst.config.ts"
     * Linkable.wrap(aws.dynamodb.Table, (table) => ({
     *   properties: { tableName: table.name },
     *   include: [
     *     sst.aws.permission({
     *       actions: ["dynamodb:*"],
     *       resources: [table.arn]
     *     })
     *   ]
     * }));
     * ```
     *
     * It's defining the properties that we want made accessible at runtime and the permissions
     * that the linked resource should have.
     *
     * Now you can link any `aws.dynamodb.Table` instances in your app just like any other SST
     * component.
     *
     * ```ts title="sst.config.ts" {7}
     * const table = new aws.dynamodb.Table("MyTable", {
     *   attributes: [{ name: "id", type: "S" }],
     *   hashKey: "id",
     * });
     *
     * new sst.aws.Nextjs("MyWeb", {
     *   link: [table]
     * });
     * ```
     *
     * Since this applies to any resource, you can also use it to wrap SST components and modify
     * how they are linked.
     *
     * ```ts title="sst.config.ts" "sst.aws.Bucket"
     * sst.Linkable.wrap(sst.aws.Bucket, (bucket) => ({
     *   properties: { name: bucket.name },
     *   include: [
     *     sst.aws.permission({
     *       actions: ["s3:GetObject"],
     *       resources: [bucket.arn]
     *     })
     *   ]
     * }));
     * ```
     *
     * This overrides the built-in link and lets you create your own.
     *
     * :::tip
     * You can modify the permissions granted by a linked resource.
     * :::
     *
     * In the above example, we're modifying the permissions to access a linked `sst.aws.Bucket`
     * in our app.
     */
    Linkable.wrap = function (cls, cb) {
        // @ts-expect-error
        this.wrappedResources.add(cls.__pulumiType);
        cls.prototype.getSSTLink = function () {
            return cb(this);
        };
    };
    Linkable.wrappedResources = new Set();
    return Linkable;
}(component_1.Component));
exports.Linkable = Linkable;
/**
 * @deprecated
 * Use sst.Linkable instead.
 */
var Resource = /** @class */ (function (_super) {
    __extends(Resource, _super);
    function Resource(name, properties) {
        var _this = _super.call(this, "sst:sst:Resource", name, {
            properties: properties,
        }, {}) || this;
        console.warn("Resource is deprecated. Use sst.Linkable instead.");
        _this._properties = properties;
        _this._name = name;
        return _this;
    }
    Object.defineProperty(Resource.prototype, "name", {
        get: function () {
            return (0, pulumi_1.output)(this._name);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Resource.prototype, "properties", {
        get: function () {
            return this._properties;
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Resource.prototype.getSSTLink = function () {
        return {
            properties: this._properties,
        };
    };
    return Resource;
}(component_1.Component));
exports.Resource = Resource;
function env(env) {
    return {
        type: "environment",
        env: env,
    };
}
