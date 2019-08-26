package dubbo.server;

import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ProtocolConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.config.ServiceConfig;
import dubbo.DubboService;
import dubbo.server.Impl.DubboServiceImpl;

import java.io.IOException;

public class Main {
    public static void main(String[] args) throws IOException {
        ApplicationConfig applicationConfig = new ApplicationConfig();
        applicationConfig.setName("java-server");

        RegistryConfig registryConfig = new RegistryConfig();
        registryConfig.setAddress("consul://127.0.0.1:8500");

        ProtocolConfig protocolConfig = new ProtocolConfig();
        protocolConfig.setName("dubbo");
        protocolConfig.setHost("127.0.0.1");
        protocolConfig.setPort(12345);

        ServiceConfig<DubboService> serviceConfig = new ServiceConfig<>();
        serviceConfig.setApplication(applicationConfig);
        serviceConfig.setRegistry(registryConfig);
        serviceConfig.setProtocol(protocolConfig);
        serviceConfig.setInterface(DubboService.class);
        serviceConfig.setRef(new DubboServiceImpl());
        serviceConfig.export();

        System.in.read();
    }
}
