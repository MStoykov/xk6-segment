import { SharedSegmentedIndex} from "k6/x/segment";

export default function() {
    var index = new SharedSegmentedIndex("some");
    var v =index.next()
    console.log(v.scaled, v.unscaled, __VU);
}
