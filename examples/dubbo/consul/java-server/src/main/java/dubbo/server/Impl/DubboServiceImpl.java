package dubbo.server.Impl;

import dubbo.DubboService;

public class DubboServiceImpl implements DubboService {

    @Override
    public String SayHello(String message) {
        return "hello " + message;
    }
}
