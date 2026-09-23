package com.microsoft.azd.springaiagent;

import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;

@SpringBootTest(properties = {
    "AZURE_INFERENCE_ENDPOINT=https://example.openai.azure.com/openai/v1",
    "AZURE_INFERENCE_MODEL=test-deployment"
})
class SpringAiTodoAgentApplicationTest {

    @Test
    void contextLoads() {
    }
}
