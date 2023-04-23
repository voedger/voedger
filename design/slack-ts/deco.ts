
function Workspace(constructor: Function) {
    Object.seal(constructor);
    Object.seal(constructor.prototype);
}

function Doc(constructor: Function) {
    Object.seal(constructor);
    Object.seal(constructor.prototype);
}


function format(formatString: string) {
    return function (target: Object, propertyKey: string) {
        let value: string;
    }
}

function OuterList(target: Object, propertyKey: string) {

}

// function OuterList() {
//     return function (target: Object, propertyKey: string) {
//         let value: string;
//     }
// }

function ListSize_Huge() {
    return function (target: Object, propertyKey: string) {
        let value: string;
    }
}

function ListSize_Small() {
    return function (target: Object, propertyKey: string) {
        let value: string;
    }
}

function Min(limit: number) {
    return function (target: Object, propertyKey: string) {
        let value: string;
        const getter = function () {
            return value;
        };
        const setter = function (newVal: string) {
            if (newVal.length < limit) {
                Object.defineProperty(target, 'errors', {
                    value: `Your password should be bigger than ${limit}`
                });
            }
            else {
                value = newVal;
            }
        };
        Object.defineProperty(target, propertyKey, {
            get: getter,
            set: setter
        });
    }
}

function color(value: string) {
    // this is the decorator factory, it sets up
    // the returned decorator function
    return function (target) {
        // this is the decorator
        // do something with 'target' and 'value'...
    };
}

function first() {
    console.log("first(): factory evaluated");
    return function (target: any, propertyKey: string, descriptor: PropertyDescriptor) {
        console.log("first(): called");
    };
}

function second() {
    console.log("second(): factory evaluated");
    return function (target: any, propertyKey: string, descriptor: PropertyDescriptor) {
        console.log("second(): called");
    };
}