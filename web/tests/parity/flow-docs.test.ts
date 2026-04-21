import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { describe, expect, it } from "vitest"

describe("flow-docs", () => {
  it("contains docs route and endpoint listings", () => {
    const docsPagePath = resolve(process.cwd(), "pages/docs.vue")
    const docsSource = readFileSync(docsPagePath, "utf8")

    expect(docsSource).toContain("API atlas")
    expect(docsSource).toContain("apiEndpoints")
  })
})
