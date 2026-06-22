import type { RouteContext } from "./context";
import type { PluginHooks } from "./plugin";
import type { AnyRouteDescriptor, RouteDescriptor } from "./route";
import type { IsAny } from "./type-utils";

/** Invoke one of the plugin's routes from `actions`. */
export type RunRoute = <I, O>(route: RouteDescriptor<I, O>, input: I) => Promise<O>;

/**
 * The models a client plugin may contribute extra fields to. Each optional key
 * maps a model name to the extra fields the plugin adds.
 */
export type PluginSchema = {
  user?: object;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- keeps RouteContext<any> assignment-compatible under AnyClientPlugin
type AnyUserFields = any;

/**
 * The `user` fields a plugin itself declared via `schema` — used to type the
 * plugin's own `ctx`, so an action can read its own contributions off
 * `ctx.store.$session` without a cast. A plugin without a `user` schema sees the
 * base context (`unknown`); widened `any` stays `any` so a concrete plugin
 * remains assignable to `AnyClientPlugin`.
 */
type OwnUserFields<Schema> =
  IsAny<Schema> extends true
    ? AnyUserFields
    : Schema extends Record<"user", infer F>
      ? F extends object
        ? F
        : unknown
      : unknown;

/**
 * Client plugin contract
 */
export type ClientPlugin<
  Id extends string,
  BasePath extends string,
  Routes extends readonly AnyRouteDescriptor[],
  Actions,
  Schema = unknown,
> = {
  readonly id: Id;
  /** Default mount path relative to the client `basePath`, e.g. `"/magic-link"`; omit for the root (`""`). */
  readonly basePath?: BasePath;
  readonly routes: Routes;
  readonly hooks?: PluginHooks;
  readonly actions?: (ctx: RouteContext<OwnUserFields<Schema>>, run: RunRoute) => Actions;
  /**
   * Type-only declaration of the extra model fields this plugin contributes to
   * the client read surfaces — folded in only when the plugin is registered.
   * Declare it with {@link schema}.
   */
  readonly schema?: Schema;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- match any plugin shape
export type AnyClientPlugin = ClientPlugin<string, string, readonly AnyRouteDescriptor[], any, any>;

/**
 * Register a plugin's routes while preserving each route's input/output types
 * for the generated client API.
 */
export function defineRoutes<Routes extends readonly AnyRouteDescriptor[]>(...routes: Routes): Routes {
  return routes;
}

/**
 * Define a client plugin for `createAuthClient`.
 */
export function defineClientPlugin<
  const Id extends string,
  const Routes extends readonly AnyRouteDescriptor[],
  Actions = Record<never, never>,
  const BasePath extends string = "",
  Schema = unknown,
>(def: ClientPlugin<Id, BasePath, Routes, Actions, Schema>): ClientPlugin<Id, BasePath, Routes, Actions, Schema> {
  return {
    id: def.id,
    basePath: (def.basePath ?? "") as BasePath,
    routes: def.routes,
    ...(def.hooks !== undefined ? { hooks: def.hooks } : {}),
    ...(def.actions !== undefined ? { actions: def.actions } : {}),
  };
}

/**
 * Declare the extra model fields a plugin contributes — type-only. The returned
 * value is a phantom (never read at runtime); actual values come from the
 * central session parse.
 *
 * @example
 *   schema: schema<{ user: { twoFactorEnabled: boolean } }>(),
 */
export function schema<T extends PluginSchema & Record<Exclude<keyof T, keyof PluginSchema>, never>>(): T {
  return undefined as unknown as T;
}
