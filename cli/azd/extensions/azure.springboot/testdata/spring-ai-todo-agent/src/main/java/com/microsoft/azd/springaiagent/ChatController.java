package com.microsoft.azd.springaiagent;

import jakarta.validation.Valid;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/chat")
public class ChatController {

    private final TodoAgentService todoAgentService;

    public ChatController(TodoAgentService todoAgentService) {
        this.todoAgentService = todoAgentService;
    }

    @PostMapping
    public ChatResponse chat(@Valid @RequestBody ChatRequest request) {
        return new ChatResponse(todoAgentService.chat(request.message()));
    }
}
