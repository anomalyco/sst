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
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Auth = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var function_1 = require("./function");
var tls_1 = require("@pulumi/tls");
var Auth = /** @class */ (function (_super) {
    __extends(Auth, _super);
    function Auth(name, args, opts) {
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        _this._key = new tls_1.PrivateKey("".concat(name, "Keypair"), {
            algorithm: "RSA",
        });
        _this._authenticator = (0, pulumi_1.output)(args.authenticator).apply(function (args) {
            return new function_1.Function("".concat(name, "Authenticator"), __assign(__assign({ url: true }, args), { environment: __assign(__assign({}, args.environment), { AUTH_PRIVATE_KEY: (0, pulumi_1.secret)(_this.key.privateKeyPemPkcs8), AUTH_PUBLIC_KEY: (0, pulumi_1.secret)(_this.key.publicKeyPem) }), _skipHint: true }));
        });
        return _this;
    }
    Object.defineProperty(Auth.prototype, "key", {
        get: function () {
            return this._key;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Auth.prototype, "authenticator", {
        get: function () {
            return this._authenticator;
        },
        enumerable: false,
        configurable: true
    });
    Object.defineProperty(Auth.prototype, "url", {
        get: function () {
            return this._authenticator.url;
        },
        enumerable: false,
        configurable: true
    });
    /** @internal */
    Auth.prototype.getSSTLink = function () {
        return {
            properties: {
                publicKey: (0, pulumi_1.secret)(this.key.publicKeyPem),
            },
        };
    };
    return Auth;
}(component_1.Component));
exports.Auth = Auth;
var __pulumiType = "sst:aws:Auth";
// @ts-expect-error
Auth.__pulumiType = __pulumiType;
