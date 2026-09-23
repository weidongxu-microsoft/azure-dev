package com.microsoft.azd.springaiagent;

import org.springframework.ai.chat.client.ChatClient;
import org.springframework.stereotype.Service;

@Service
public class TodoAgentService {

    private static final String SYSTEM_PROMPT = """
        You are a concise todo assistant. Use the available tools to list, create, or complete todo items.
        Do not claim that a todo operation succeeded unless its tool call succeeded.
        """;

    private final ChatClient chatClient;
    private final TodoTools todoTools;

    public TodoAgentService(ChatClient.Builder chatClientBuilder, TodoTools todoTools) {
        this.chatClient = chatClientBuilder
            .defaultSystem(SYSTEM_PROMPT)
            .build();
        this.todoTools = todoTools;
    }

    public String chat(String message) {
        return chatClient.prompt()
            .user(message)
            .tools(todoTools)
            .call()
            .content();
    }
}
