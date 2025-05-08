"use strict";
// More examples here on how to parse responses from Cloudflare API
// https://github.com/cloudflare/workers-sdk/blob/main/packages/wrangler/src/routes.ts
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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
var _a, _b, _c;
Object.defineProperty(exports, "__esModule", { value: true });
exports.cfFetch = cfFetch;
var CLOUDFLARE_API_BASE_URL = (_a = process.env.CLOUDFLARE_API_BASE_URL) !== null && _a !== void 0 ? _a : "https://api.cloudflare.com/client/v4";
var CLOUDFLARE_API_TOKEN = ((_c = (_b = $app.providers) === null || _b === void 0 ? void 0 : _b.cloudflare) === null || _c === void 0 ? void 0 : _c.apiToken) || process.env.CLOUDFLARE_API_TOKEN;
function cfFetch(resource_1) {
    return __awaiter(this, arguments, void 0, function (resource, init) {
        var ret, json, error;
        if (init === void 0) { init = {}; }
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0: return [4 /*yield*/, fetch("".concat(CLOUDFLARE_API_BASE_URL).concat(resource), __assign(__assign({}, init), { headers: __assign({ Authorization: "Bearer ".concat(CLOUDFLARE_API_TOKEN) }, init.headers) }))];
                case 1:
                    ret = _a.sent();
                    return [4 /*yield*/, ret.json()];
                case 2:
                    json = (_a.sent());
                    // ie.
                    // {
                    //   "result": {
                    //     "subdomain": "wangfanjie"
                    //   },
                    //   "success": true,
                    //   "errors": [],
                    //   "messages": []
                    // }
                    if (json.success) {
                        return [2 /*return*/, json];
                    }
                    error = new Error("A request to the Cloudflare API (".concat(resource, ") failed."));
                    // @ts-expect-error attach the errors to the error object
                    error.errors = json.errors;
                    // @ts-expect-error attach the messages to the error object
                    error.messages = json.messages;
                    throw error;
            }
        });
    });
}
