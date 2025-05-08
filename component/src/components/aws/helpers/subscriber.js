"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isFunctionSubscriber = isFunctionSubscriber;
exports.isQueueSubscriber = isQueueSubscriber;
var pulumi_1 = require("@pulumi/pulumi");
var queue_1 = require("../queue");
function isFunctionSubscriber(subscriber) {
    if (!subscriber)
        return (0, pulumi_1.output)(false);
    return (0, pulumi_1.output)(subscriber).apply(function (subscriber) {
        return typeof subscriber === "string" || typeof subscriber.handler === "string";
    });
}
function isQueueSubscriber(subscriber) {
    if (!subscriber)
        return (0, pulumi_1.output)(false);
    return (0, pulumi_1.output)(subscriber).apply(function (subscriber) {
        return typeof subscriber === "string" || subscriber instanceof queue_1.Queue;
    });
}
