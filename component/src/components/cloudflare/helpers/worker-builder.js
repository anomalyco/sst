"use strict";
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
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.workerBuilder = workerBuilder;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../../component");
var worker_1 = require("../worker");
function workerBuilder(name, definition, argsTransform, opts) {
    return (0, pulumi_1.output)(definition).apply(function (definition) {
        if (typeof definition === "string") {
            // Case 1: The definition is a handler
            var worker_2 = new (worker_1.Worker.bind.apply(worker_1.Worker, __spreadArray([void 0], (0, component_1.transform)(argsTransform, name, { handler: definition }, opts || {}), false)))();
            return {
                getWorker: function () { return worker_2; },
                script: worker_2.nodes.worker,
            };
        }
        // Case 2: The definition is a WorkerArgs
        else if (definition.handler) {
            var worker_3 = new (worker_1.Worker.bind.apply(worker_1.Worker, __spreadArray([void 0], (0, component_1.transform)(argsTransform, name, __assign({}, definition), opts || {}), false)))();
            return {
                getWorker: function () { return worker_3; },
                script: worker_3.nodes.worker,
            };
        }
        throw new Error("Invalid worker definition for the \"".concat(name, "\" Worker"));
    });
}
