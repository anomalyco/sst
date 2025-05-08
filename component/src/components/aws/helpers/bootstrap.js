"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.bootstrap = void 0;
var rpc_1 = require("../../rpc/rpc");
exports.bootstrap = {
    forRegion: function (region) {
        return rpc_1.rpc.call("Provider.Aws.Bootstrap", { region: region });
    },
};
