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

const createAnalyticsTopics = (scope: string, entity: string) => ({
  snapshot: `analytics-${scope}-${entity}.snapshot` as const,
  error: `analytics-${scope}-${entity}.error` as const,
});

const ANALYTICS_TOPICS = {
  public: {
    users: createAnalyticsTopics("public", "users"),
    dishes: createAnalyticsTopics("public", "dishes"),
    menus: createAnalyticsTopics("public", "menus"),
  },
  restaurant: {
    users: createAnalyticsTopics("restaurant", "users"),
  },
  admin: {
    auth: createAnalyticsTopics("admin", "auth"),
    restaurants: createAnalyticsTopics("admin", "restaurants"),
    sections: createAnalyticsTopics("admin", "sections"),
    tables: createAnalyticsTopics("admin", "tables"),
    images: createAnalyticsTopics("admin", "images"),
    objects: createAnalyticsTopics("admin", "objects"),
    subscriptions: createAnalyticsTopics("admin", "subscriptions"),
    "subscription-plans": createAnalyticsTopics("admin", "subscription-plans"),
    reservations: createAnalyticsTopics("admin", "reservations"),
    reviews: createAnalyticsTopics("admin", "reviews"),
    payments: createAnalyticsTopics("admin", "payments"),
  },
} as const;

const ANALYTICS_COMMANDS = {
  refresh: "command.analytics.refresh" as const,
  fetch: "command.analytics.fetch" as const,
  query: "command.analytics.query" as const,
} as const;

export const WEBSOCKET_TOPICS = {
  system: {
    connected: "system.connected",
    pong: "system.pong",
    error: "system.error",
  },
  analytics: ANALYTICS_TOPICS,
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
  analytics: ANALYTICS_COMMANDS,
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

export const WEBSOCKET_EXAMPLES = {
  realtime: {
    restaurantsList: {
      description: "Snapshot paginado de restaurantes de una sección",
      url: "ws://localhost:8080/ws/restaurants/main-hall",
      command: {
        action: "list_restaurants",
        payload: {
          page: 1,
          limit: 20,
          search: "",
        },
      },
    },
    tablesDetail: {
      description: "Detalle puntual de una mesa en la sección principal",
      url: "ws://localhost:8080/ws/tables/main-hall",
      command: {
        action: "get_table",
        payload: {
          id: "table-17",
        },
      },
    },
  },
  analytics: {
    publicUsers: {
      description: "Tendencias públicas de usuarios sin token",
      url: "ws://localhost:8080/ws/analytics/public/users",
      command: {
        action: "fetch",
        payload: {
          query: {
            startDate: "2024-01-01",
          },
        },
      },
    },
    restaurantUsers: {
      description: "Indicadores de usuarios por restaurante (JWT requerido)",
      url: "ws://localhost:8080/ws/analytics/restaurant/users?restaurantId=rest-123",
      command: {
        action: "refresh",
        payload: {
          identifier: "rest-123",
          query: {
            startDate: "2024-01-01",
          },
        },
      },
    },
    adminPayments: {
      description: "Panel administrativo de pagos filtrado por restaurante",
      url: "ws://localhost:8080/ws/analytics/admin/payments?restaurantId=rest-123&startDate=2024-01-01",
      command: {
        action: "query",
        payload: {
          query: {
            restaurantId: "rest-123",
            startDate: "2024-01-01",
          },
        },
      },
    },
  },
} as const;

type TopicTree<T> = T extends string ? T : TopicTree<T[keyof T]>;

type CommandTree<T> = T extends string ? T : CommandTree<T[keyof T]>;

export type WebsocketEntity = keyof typeof WEBSOCKET_TOPICS;
export type WebsocketTopic = TopicTree<typeof WEBSOCKET_TOPICS>;
export type WebsocketCommand = CommandTree<typeof WEBSOCKET_COMMANDS>;
