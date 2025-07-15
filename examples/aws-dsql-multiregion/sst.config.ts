/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-dsql-multiregion",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create clusters in different regions
    // Note: Both clusters need the same witnessRegion for multi-region setup
    const witnessRegion = "us-west-2";
    
    const primary = new sst.aws.Dsql("Primary", {
      deletionProtection: false, // Required for deletion
      witnessRegion: witnessRegion, // Required for multi-region
    });

    // Create peer cluster in different region
    const peerProvider = new aws.Provider("PeerRegion", { 
      region: "us-east-2" 
    });
    
    const peer = new sst.aws.Dsql("Peer", {
      deletionProtection: false, // Required for deletion
      witnessRegion: witnessRegion, // Required for multi-region
    }, { 
      provider: peerProvider 
    });

    // Create peering between clusters
    const peering = new sst.aws.DsqlPeering("Peering", {
      primaryCluster: primary,
      peerCluster: peer,
      witnessRegion: witnessRegion,
      tags: {
        Environment: $app.stage,
        Example: "aws-dsql-multiregion",
      },
    });

    // Create a function that can connect to either cluster
    const fn = new sst.aws.Function("MyFunction", {
      handler: "src/lambda.handler",
      link: [primary, peer],
    });

    return {
      primary: {
        arn: primary.arn,
        identifier: primary.identifier,
        endpoint: primary.publicEndpoint,
        region: primary.region,
      },
      peer: {
        arn: peer.arn,
        identifier: peer.identifier,
        endpoint: peer.publicEndpoint,
        region: peer.region,
      },
      peering: {
        id: peering.peeringId,
        witnessRegion: peering.witnessRegion,
      },
      function: fn.arn,
    };
  },
});
