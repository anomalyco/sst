# Document Component

Steps to add a new component to the SST docs.

1. Register in docs generator
   - Add the component `.ts` path to `entryPoints` in `www/generate.ts`

2. Add matching public getters
   - Ensure every property returned by `getSSTLink()` has a corresponding public getter on the class
   - The generator crashes otherwise because `renderLinks()` looks for a getter with the same name

3. Analyze docs for relevant places to link the component
   - Search existing guides (e.g. `cloudflare.mdx`) and component overviews for lists of related components
   - Add the new component link where appropriate

4. Add to Astro sidebar
   - Add the generated doc slug to the appropriate section in `www/astro.config.mjs`
   - Follow the existing visual ordering: components are sorted from shortest to longest name (least to most characters)

5. Generate docs
   - Run `bun run docs:generate`

