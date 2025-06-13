#!/usr/bin/env bun

import { $ } from "bun";

await $`rm -rf ./dist`;

await $`tsc`;