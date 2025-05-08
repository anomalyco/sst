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
exports.Secret = exports.SecretMissingError = void 0;
var error_1 = require("./error");
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("./component");
var SecretMissingError = /** @class */ (function (_super) {
    __extends(SecretMissingError, _super);
    function SecretMissingError(secretName) {
        var _this = _super.call(this, "Set a value for ".concat(secretName, " with `sst secret set ").concat(secretName, " <value>`")) || this;
        _this.secretName = secretName;
        return _this;
    }
    return SecretMissingError;
}(error_1.VisibleError));
exports.SecretMissingError = SecretMissingError;
/**
 * The `Secret` component lets you create secrets in your app.
 *
 * <VideoAside title="Watch a video on how secrets work" href="https://youtu.be/7tW2L3P6LKw" />
 *
 * Secrets are encrypted and stored in an S3 Bucket in your AWS account. If used in
 * your app config, they'll be encrypted in your state file as well. If used in
 * your function code, they are encrypted and included in the bundle. They'll are
 * then decrypted synchronously when your function starts up by the SST SDK.
 *
 * @example
 *
 * #### Create a secret
 *
 * The name of a secret follows the same rules as a component name. It must start with a capital letter and contain only letters and numbers.
 *
 * :::note
 * Secret names must start with a capital letter and contain only letters and numbers.
 * :::
 *
 * ```ts title="sst.config.ts"
 * const secret = new sst.Secret("MySecret");
 * ```
 *
 * #### Set a placeholder
 *
 * You can optionally set a `placeholder`.
 *
 * :::tip
 * Useful for cases where you might use a secret for values that aren't sensitive, so you can just set them in code.
 * :::
 *
 * ```ts title="sst.config.ts"
 * const secret = new sst.Secret("MySecret", "my-secret-placeholder-value");
 * ```
 *
 * #### Set the value of the secret
 *
 * You can then set the value of a secret using the [CLI](/docs/reference/cli/).
 *
 * ```sh title="Terminal"
 * sst secret set MySecret my-secret-value
 * ```
 *
 * :::note
 * If you are not running `sst dev`, you'll need to `sst deploy` to apply the secret.
 * :::
 *
 * #### Set a fallback for the secret
 *
 * You can set a _fallback_ value for the secret with the `--fallback` flag. If the secret is
 * not set for a stage, it'll use the fallback value instead.
 *
 * ```sh title="Terminal"
 * sst secret set MySecret my-fallback-value --fallback
 * ```
 *
 * This is useful for PR environments that are auto-deployed.
 *
 * #### Use the secret in your app config
 *
 * You can now use the secret in your app config.
 *
 * ```ts title="sst.config.ts"
 * console.log(mySecret.value);
 * ```
 *
 * This is an [Output](/docs/components#outputs) that can be used as an Input to other components.
 *
 * #### Link the secret to a resource
 *
 * You can link the secret to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [secret]
 * });
 * ```
 *
 * Once linked, you can use the secret in your function code.
 *
 * ```ts title="app/page.tsx"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MySecret.value);
 * ```
 */
var Secret = /** @class */ (function (_super) {
    __extends(Secret, _super);
    /**
     * @param placeholder A placeholder value of the secret. This can be useful for cases where you might not be storing sensitive values.
  
     */
    function Secret(name, placeholder) {
        var _a;
        var _this = _super.call(this, "sst:sst:Secret", name, {
            placeholder: placeholder,
        }, {}) || this;
        _this._name = name;
        _this._placeholder = placeholder ? (0, pulumi_1.output)(placeholder) : undefined;
        _this._value = (0, pulumi_1.output)((_a = process.env["SST_SECRET_" + _this._name]) !== null && _a !== void 0 ? _a : _this._placeholder).apply(function (value) {
            if (typeof value !== "string") {
                throw new SecretMissingError(_this._name);
            }
            return value;
        });
        return _this;
    }
    Object.defineProperty(Secret.prototype, "name", {
        /**
         * The name of the secret.
         */
        get: function () {
            return (0, pulumi_1.output)(this._name);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Secret.prototype, "value", {
        /**
         * The value of the secret. It'll be `undefined` if the secret has not been set through the CLI or if the `placeholder` hasn't been set.
         */
        get: function () {
            return (0, pulumi_1.secret)(this._value);
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Secret.prototype, "placeholder", {
        /**
         * The placeholder value of the secret.
         */
        get: function () {
            return this._placeholder;
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Secret.prototype.getSSTLink = function () {
        return {
            properties: {
                value: this.value,
            },
        };
    };
    return Secret;
}(component_1.Component));
exports.Secret = Secret;
