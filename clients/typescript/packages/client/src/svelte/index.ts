import type { Readable } from "svelte/store";
import { createAuthClient as createCoreClient } from "../client";
import type { AnyClientPlugin } from "../define-plugin";
import type { InferUserFields } from "../infer";
import type { SessionState } from "../session-store";
import type { Prettify } from "../type-utils";
import type { AuthClient, CreateAuthClientOptions, PrettyUserFields } from "../types";

/**
 * An {@link AuthClient} augmented with Svelte stores.
 */
export type SvelteAuthClient<Plugins extends readonly AnyClientPlugin[], TFields = unknown> = Prettify<
  AuthClient<Plugins, TFields> & {
    /**
     * The reactive session as a Svelte readable store. Use it with `$`:
     * `$session.data`, `$session.isPending`, `$session.error`.
     */
    useSession: () => Readable<SessionState<PrettyUserFields<Plugins, TFields>>>;
  }
>;

/**
 * Create a Limen auth client with Svelte stores attached.
 */
export function createAuthClient<const Plugins extends readonly AnyClientPlugin[] = readonly [], TFields = unknown>(
  opts: CreateAuthClientOptions<Plugins, TFields>,
): SvelteAuthClient<Plugins, TFields> {
  const client = createCoreClient<Plugins, TFields>(opts);
  const useSession = (): Readable<SessionState<PrettyUserFields<Plugins, TFields>>> =>
    client.$session as unknown as Readable<SessionState<InferUserFields<Plugins, TFields>>>;

  return Object.assign(client, { useSession }) as SvelteAuthClient<Plugins, TFields>;
}

export type { SessionState, SessionStore } from "../session-store";
export type { AuthClient, CreateAuthClientOptions, Session, User } from "../types";
