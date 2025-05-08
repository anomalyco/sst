"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.iamEdit = exports.permission = exports.linkable = void 0;
__exportStar(require("./analog.js"), exports);
__exportStar(require("./apigatewayv1.js"), exports);
__exportStar(require("./apigatewayv2.js"), exports);
__exportStar(require("./apigateway-websocket.js"), exports);
__exportStar(require("./app-sync.js"), exports);
__exportStar(require("./astro.js"), exports);
__exportStar(require("./aurora.js"), exports);
__exportStar(require("./auth.js"), exports);
__exportStar(require("./bucket.js"), exports);
__exportStar(require("./bus.js"), exports);
__exportStar(require("./cluster.js"), exports);
__exportStar(require("./cognito-identity-pool.js"), exports);
__exportStar(require("./cognito-user-pool.js"), exports);
__exportStar(require("./cron.js"), exports);
__exportStar(require("./dns.js"), exports);
__exportStar(require("./dynamo.js"), exports);
__exportStar(require("./efs.js"), exports);
__exportStar(require("./email.js"), exports);
__exportStar(require("./function.js"), exports);
__exportStar(require("./kinesis-stream.js"), exports);
__exportStar(require("./nextjs.js"), exports);
__exportStar(require("./opencontrol.js"), exports);
__exportStar(require("./open-search.js"), exports);
__exportStar(require("./postgres.js"), exports);
__exportStar(require("./mysql.js"), exports);
__exportStar(require("./queue.js"), exports);
__exportStar(require("./realtime.js"), exports);
__exportStar(require("./react.js"), exports);
__exportStar(require("./redis.js"), exports);
__exportStar(require("./remix.js"), exports);
__exportStar(require("./router.js"), exports);
__exportStar(require("./service.js"), exports);
__exportStar(require("./sns-topic.js"), exports);
__exportStar(require("./solid-start.js"), exports);
__exportStar(require("./tan-stack-start.js"), exports);
__exportStar(require("./task.js"), exports);
__exportStar(require("./nuxt.js"), exports);
__exportStar(require("./static-site.js"), exports);
__exportStar(require("./svelte-kit.js"), exports);
__exportStar(require("./vector.js"), exports);
__exportStar(require("./vpc.js"), exports);
var linkable_js_1 = require("./linkable.js");
Object.defineProperty(exports, "linkable", { enumerable: true, get: function () { return linkable_js_1.linkable; } });
var permission_js_1 = require("./permission.js");
Object.defineProperty(exports, "permission", { enumerable: true, get: function () { return permission_js_1.permission; } });
var iam_edit_js_1 = require("./iam-edit.js");
Object.defineProperty(exports, "iamEdit", { enumerable: true, get: function () { return iam_edit_js_1.iamEdit; } });
// internal components
__exportStar(require("./cdn.js"), exports);
__exportStar(require("./dns-validated-certificate.js"), exports);
