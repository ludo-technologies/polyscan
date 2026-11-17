package com.example.app;

import com.example.model.*;
import com.example.service.Service;
import java.util.List;

public class App {
    public static void main(String[] args) {
        List<Shape> shapes = new Service().shapes();
        System.out.println(shapes.size());
    }
}
