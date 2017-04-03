/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * THIS FILE WAS GENERATED BY codergen. EDIT WITH CARE.
 */
package com.google.gapid.rpclib.any;

import com.google.gapid.rpclib.binary.*;
import com.google.gapid.rpclib.schema.*;

import java.io.IOException;

final class Int8 extends Box implements BinaryObject {
    @Override
    public Object unwrap() {
        return getValue();
    }

    //<<<Start:Java.ClassBody:1>>>
    private byte mValue;

    // Constructs a default-initialized {@link Int8}.
    public Int8() {}


    public byte getValue() {
        return mValue;
    }

    public Int8 setValue(byte v) {
        mValue = v;
        return this;
    }

    @Override
    public BinaryClass klass() { return Klass.INSTANCE; }


    private static final Entity ENTITY = new Entity("any", "int8_", "", "");

    static {
        ENTITY.setFields(new Field[]{
            new Field("Value", new Primitive("int8", Method.Int8)),
        });
        Namespace.register(Klass.INSTANCE);
    }
    public static void register() {}
    //<<<End:Java.ClassBody:1>>>
    public enum Klass implements BinaryClass {
        //<<<Start:Java.KlassBody:2>>>
        INSTANCE;

        @Override
        public Entity entity() { return ENTITY; }

        @Override
        public BinaryObject create() { return new Int8(); }

        @Override
        public void encode(Encoder e, BinaryObject obj) throws IOException {
            Int8 o = (Int8)obj;
            e.int8(o.mValue);
        }

        @Override
        public void decode(Decoder d, BinaryObject obj) throws IOException {
            Int8 o = (Int8)obj;
            o.mValue = d.int8();
        }
        //<<<End:Java.KlassBody:2>>>
    }
}
