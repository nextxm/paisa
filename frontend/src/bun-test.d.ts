declare module "bun:test" {
  export function describe(name: string, fn: () => void): void;
  export function test(name: string, fn: () => void | Promise<void>): void;
  export function beforeEach(fn: () => void | Promise<void>): void;
  export function afterEach(fn: () => void | Promise<void>): void;

  export const mock: {
    <T extends (...args: any[]) => any>(fn: T): T;
    module(specifier: string, factory: () => Record<string, unknown>): void;
  };

  interface Matchers<T> {
    toBe(expected: unknown): void;
    toBeCloseTo(expected: number, numDigits?: number): void;
    toEqual(expected: unknown): void;
    toContain(expected: unknown): void;
    toBeGreaterThan(expected: number): void;
    toBeLessThan(expected: number): void;
    toThrow(expected?: string | RegExp): void;
    resolves: Matchers<T>;
    rejects: Matchers<T>;
  }

  export function expect<T>(actual: T): Matchers<T>;
}
