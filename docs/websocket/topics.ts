// Central catalog of WebSocket topics and commands to mirror the Kafka contract.
const createEntityTopics = <T extends string>(entity: T) => ({
  snapshot: `${entity}.snapshot` as const,
  list: `${entity}.list` as const,
  detail: `${entity}.detail` as const,
  error: `${entity}.error` as const,
  created: `${entity}.created` as const,
  updated: `${entity}.updated` as const,
  deleted: `${entity}.deleted` as const,
});

const createEntityCommands = <Plural extends string, Singular extends string>(
  plural: Plural,
  singular: Singular
) => ({
  list: `command.list_${plural}` as const,
  get: `command.get_${singular}` as const,
});

export const WEBSOCKET_TOPICS = {
  system: {
    connected: "system.connected",
    pong: "system.pong",
    error: "system.error",
  },
  reviews: createEntityTopics("reviews"),
  restaurants: createEntityTopics("restaurants"),
  sections: createEntityTopics("sections"),
  tables: createEntityTopics("tables"),
  objects: createEntityTopics("objects"),
  menus: createEntityTopics("menus"),
  dishes: createEntityTopics("dishes"),
  images: createEntityTopics("images"),
  "section-objects": createEntityTopics("section-objects"),
  reservations: createEntityTopics("reservations"),
  payments: createEntityTopics("payments"),
  subscriptions: createEntityTopics("subscriptions"),
  "subscription-plans": createEntityTopics("subscription-plans"),
  "auth-users": createEntityTopics("auth-users"),
} as const;

export const WEBSOCKET_COMMANDS = {
  system: {
    ping: "command.ping",
    subscribe: "command.subscribe",
    unsubscribe: "command.unsubscribe",
  },
  reviews: createEntityCommands("reviews", "review"),
  restaurants: createEntityCommands("restaurants", "restaurant"),
  sections: createEntityCommands("sections", "section"),
  tables: createEntityCommands("tables", "table"),
  objects: createEntityCommands("objects", "object"),
  menus: createEntityCommands("menus", "menu"),
  dishes: createEntityCommands("dishes", "dish"),
  images: createEntityCommands("images", "image"),
  "section-objects": createEntityCommands("section_objects", "section_object"),
  reservations: createEntityCommands("reservations", "reservation"),
  payments: createEntityCommands("payments", "payment"),
  subscriptions: createEntityCommands("subscriptions", "subscription"),
  "subscription-plans": createEntityCommands(
    "subscription_plans",
    "subscription_plan"
  ),
  "auth-users": createEntityCommands("auth_users", "auth_user"),
} as const;

type TopicTree<T> = T extends string ? T : TopicTree<T[keyof T]>;

type CommandTree<T> = T extends string ? T : CommandTree<T[keyof T]>;

export type WebsocketEntity = keyof typeof WEBSOCKET_TOPICS;
export type WebsocketTopic = TopicTree<typeof WEBSOCKET_TOPICS>;
export type WebsocketCommand = CommandTree<typeof WEBSOCKET_COMMANDS>;
