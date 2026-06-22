import type { ClientPlugin, PluginSchema } from "./define-plugin";
import type { InputOf, OutputOf, RouteCallOptions } from "./route";
import type { IsAny, KebabToCamel, Split, UnionToIntersection } from "./type-utils";

type IsParam<S extends string> = S extends `:${string}` ? true : false;

/**
 * Turn a route path literal into its camelCased client-chain segments, dropping
 * leading slashes and `:param` segments. `"/otp/send"` -> `["otp", "send"]`.
 */
export type PathSegments<P extends string> = P extends `/${infer Rest}`
  ? PathSegments<Rest>
  : P extends `${infer Head}/${infer Tail}`
    ? Head extends ""
      ? PathSegments<Tail>
      : IsParam<Head> extends true
        ? PathSegments<Tail>
        : [KebabToCamel<Head>, ...PathSegments<Tail>]
    : P extends ""
      ? []
      : IsParam<P> extends true
        ? []
        : [KebabToCamel<P>];

type PathOf<R> = R extends { path: infer P extends string } ? P : never;

/**
 * The resolved chain for one route: an absolute `as` wins, otherwise it is the
 * plugin's base-path segments followed by the route-path segments.
 */
type ChainSegments<R, BasePrefix extends readonly string[]> = R extends { as: infer A extends string }
  ? Split<A, ".">
  : [...BasePrefix, ...PathSegments<PathOf<R>>];

type Nest<Segs extends readonly string[], Fn> = Segs extends readonly [
  infer Head extends string,
  ...infer Rest extends readonly string[],
]
  ? // Avoid emitting an index signature when route paths widen to `string`.
    string extends Head
    ? unknown
    : { [K in Head]: Nest<Rest, Fn> }
  : Fn;

/**
 * A no-input route (`I` is `void`) takes no input — only the optional, trailing
 * call options; everything else takes `input` followed by the call options.
 * Options are always the trailing argument so the runtime can place them
 * unambiguously (a lone argument is always the route input).
 */
type InferRouteFn<I, O> = [I] extends [void]
  ? (input?: void, opts?: RouteCallOptions<O>) => Promise<O>
  : (input: I, opts?: RouteCallOptions<O>) => Promise<O>;

type IsExposed<R> = R extends { expose: false } ? false : true;

type RouteFn<R> = InferRouteFn<InputOf<R>, OutputOf<R>>;

type InferOneRoute<R, BasePrefix extends readonly string[]> =
  IsExposed<R> extends false ? unknown : Nest<ChainSegments<R, BasePrefix>, RouteFn<R>>;

/**
 * Infer the public API tree for a list of route descriptors, given the owning
 * plugin's base-path segments. Each route mounts a callable at its resolved
 * chain; sibling keys merge via intersection, so `signin.credential` and
 * `signup.credential` coexist and both autocomplete.
 */
export type InferRoutes<Routes extends readonly unknown[], BasePrefix extends readonly string[]> = UnionToIntersection<
  { [K in keyof Routes]: InferOneRoute<Routes[K], BasePrefix> }[number]
>;

/**
 * The client-only methods a plugin contributes. Widened `any` contributes
 * nothing so the client does not become permissive.
 */
type ActionsOf<P> = P extends { actions?: (ctx: never, run: never) => infer A }
  ? IsAny<A> extends true
    ? unknown
    : A extends object
      ? A
      : unknown
  : unknown;

export type InferPluginContribution<P> =
  P extends ClientPlugin<infer _Id, infer BasePath, infer Routes, infer _Actions, infer _Schema>
    ? InferRoutes<Routes, PathSegments<BasePath>> & ActionsOf<P>
    : unknown;

export type CombinedClientContributions<Plugins extends readonly unknown[]> = UnionToIntersection<
  { [K in keyof Plugins]: InferPluginContribution<Plugins[K]> }[number]
>;

/**
 * The extra fields a plugin contributes to one model (`M`) via `schema`. Widened
 * `any` and non-object declarations contribute nothing, so the folded type never
 * collapses into an open index signature.
 */
type ModelFieldsOf<P, M extends keyof PluginSchema> = P extends { schema?: infer S }
  ? IsAny<S> extends true
    ? Record<never, never>
    : S extends Record<M, infer F>
      ? F extends object
        ? F
        : Record<never, never>
      : Record<never, never>
  : Record<never, never>;

/** Intersection of every registered plugin's contributions to one model. */
export type PluginModelFields<Plugins extends readonly unknown[], M extends keyof PluginSchema> = UnionToIntersection<
  { [K in keyof Plugins]: ModelFieldsOf<Plugins[K], M> }[number]
>;

/** A consumer's `TFields` widened with all plugin-contributed `user` fields. */
export type InferUserFields<Plugins extends readonly unknown[], TFields> = TFields & PluginModelFields<Plugins, "user">;
