
### **Prompt para Asistente de IA: Implementación Eficiente de WebSocket en Go para "MesaYa"**

#### **1. Contexto General del Proyecto**

Estoy desarrollando una aplicación llamada **"MesaYa"**. La arquitectura se compone de dos microservicios principales:

1.  **API Principal (Backend Nest.js):** Una API RESTful que gestiona toda la lógica de negocio (autenticación, restaurantes, secciones, mesas, etc.). Actúa como el **productor** de eventos para Kafka.
2.  **Servicio de Realtime (Backend Go):** Su único propósito es gestionar conexiones WebSocket para enviar actualizaciones en tiempo real a los clientes, actuando como **consumidor** de los eventos de Kafka.

La comunicación entre ambos es asíncrona a través de **Apache Kafka**.

#### **2. Objetivo Principal y Crítico**

Quiero que implementes una lógica **altamente eficiente** en el servicio de **Go** para las actualizaciones en tiempo real de las **secciones de un restaurante**. El flujo debe ser el siguiente:

1.  Un cliente se conecta vía WebSocket y recibe el estado **completo e inicial** de una sección mediante **una única llamada a la API REST**.
2.  Después de eso, **todas las actualizaciones** para esa sección deben llegar **exclusivamente a través de eventos de Kafka**. El servicio de Go **no debe volver a llamar a la API REST** para obtener actualizaciones; debe confiar en la información contenida en los mensajes de Kafka.

La conexión se establece a través de la URL: `/ws/restaurant/:sectionId/:jwtToken`.

#### **3. Arquitectura y Flujo de Datos Actual**

  * **API Nest.js:**

      * Usa `swagger.yml` / `swagger.json` para definir sus endpoints.
      * La autenticación se basa en JWT con RBAC, detallado en `AUTH_README.md`.
      * Publica eventos en Kafka para operaciones CRUD. Crucialmente, estos eventos contienen la **entidad completa** en el payload, incluso en eliminaciones, como se especifica en `kafka-guide.md`.

  * **Servicio Realtime (Go):**

      * Debe seguir los principios de **Clean Architecture** definidos en `CLEAN_ARCHITECTURE.md`.
      * Utiliza el framework **Echo** para HTTP y la librería **Gorilla** para WebSockets.

#### **4. Requisitos Detallados para la Implementación en Go**

Basado en los archivos de referencia, implementa el siguiente comportamiento en el servicio de Go:

**Requisito 1: Gestión de la Conexión WebSocket**
Cuando un cliente se conecta a `/ws/restaurant/:sectionId/:jwtToken`:

1.  **Validación del Token:** Extrae y valida el `jwtToken` de la URL usando el secreto `JWT_SECRET`.
2.  **Autorización:** Decodifica el token para obtener el contexto del usuario (`userId`, `roles`) y asegúrate de que tiene permiso para acceder a la `sectionId`.
3.  **Registro del Cliente:** Si es exitoso, registra al cliente en el `Hub` de WebSockets, asociándolo explícitamente a la `sectionId`. El `Hub` debe mapear cada `sectionId` a los clientes que la están observando.

-----

**Requisito 2: Envío del "Snapshot" Inicial (Una Sola Vez)**
Esta es la **única vez** que se debe contactar a la API de Nest.js para obtener el estado de una sección.

1.  **Llamada a la API de Nest.js:** Inmediatamente después de una conexión exitosa, realiza **una única petición `GET`** al endpoint `/api/section/{id}`. Usa el `jwtToken` del cliente para autenticar esta llamada.
2.  **Enviar el Snapshot:** Envía el estado completo de la sección (con sus mesas y objetos) al cliente recién conectado. Este es su estado base. El mensaje debe tener un formato claro como:
    ```json
    {
      "event": "section.snapshot",
      "payload": { ...datos completos de la sección... }
    }
    ```

-----

**Requisito 3: Actualizaciones Basadas Pura y Exclusivamente en Kafka**
Este es el núcleo de la eficiencia. El servicio **no debe hacer más llamadas REST** después del snapshot inicial.

1.  **Consumir el Mensaje de Kafka:** Escucha los tópicos relevantes (ej: `mesa-ya.tables.updated`, `mesa-ya.objects.deleted`, etc.).
2.  **Usar el Payload del Evento:** Al recibir un mensaje, extrae la entidad **directamente del payload del evento**. La documentación en `kafka-guide.md` confirma que la entidad completa ya está ahí, por lo que no se necesita información adicional.
3.  **Identificar la Sección de Destino:** Extrae la `sectionId` de la entidad dentro del payload del evento.
4.  **Broadcast Inteligente y Dirigido:** Usa el `Hub` para encontrar a todos los clientes suscritos a esa `sectionId` específica.
5.  **Enviar la Actualización:** Envía la actualización (el payload del evento de Kafka) **únicamente** a los clientes afectados. El formato debe ser consistente:
    ```json
    {
      "event": "table.updated",
      "payload": { ...entidad de la mesa del evento de Kafka... }
    }
    ```

-----

**Requisito 4: Adherencia a Clean Architecture**
Asegúrate de que la implementación siga las guías de `CLEAN_ARCHITECTURE.md`:

  * La lógica de negocio y las llamadas a servicios externos deben estar en las capas `application` e `infrastructure`, no en `transport`.
  * Define `ports` (interfaces) en la capa de `application` para abstraer las dependencias externas como el cliente HTTP que llama a Nest.js.